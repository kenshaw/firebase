// Package firebase provides a Firebase v3.0.0+ compatible API implementation.
package firebase

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/knq/oauth2util"
)

const (
	// DefaultWatchBuffer is the default length of an event channel created on
	// a call to Watch.
	DefaultWatchBuffer = 64
)

// Ref is a Firebase database reference.
type Ref struct {
	rw sync.RWMutex

	url       *url.URL
	transport http.RoundTripper

	// oauth2 token
	auth   *oauth2util.JwtBearerToken
	source oauth2.TokenSource

	queryOpts []QueryOption

	watchBufLen int
}

// NewDatabaseRef creates a new Firebase base ref using the supplied options.
func NewDatabaseRef(opts ...Option) (*Ref, error) {
	var err error

	// create client
	r := &Ref{
		watchBufLen: DefaultWatchBuffer,
	}

	// apply opts
	for _, o := range opts {
		err = o(r)
		if err != nil {
			return nil, err
		}
	}

	// check url was set
	if r.url == nil {
		return nil, errors.New("no firebase url specified")
	}

	return r, nil
}

// httpClient returns a http.Client suitable for use with Firebase.
func (r *Ref) httpClient() (*http.Client, error) {
	r.rw.RLock()
	defer r.rw.RUnlock()

	transport := r.transport

	// set oauth2 transport
	if r.source != nil {
		transport = &oauth2.Transport{
			Source: r.source,
			Base:   transport,
		}
	}

	return &http.Client{
		Transport: transport,
	}, nil
}

// createRequest creates a http.Request for the Firebase ref with method and
// supplied query options.
func (r *Ref) createRequest(method string, body io.Reader, opts ...QueryOption) (*http.Request, error) {
	var err error

	// build url
	u := r.URL().String() + ".json"

	if len(r.queryOpts) > 0 {
		opts = append(r.queryOpts, opts...)
	}

	// build query params
	if len(opts) > 0 {
		v := make(url.Values)
		for _, o := range opts {
			err = o(v)
			if err != nil {
				return nil, err
			}
		}

		if vstr := v.Encode(); vstr != "" {
			u = u + "?" + vstr
		}
	}

	// create request
	return http.NewRequest(method, u, body)
}

// clientAndRequest creates a *http.Client and *http.Request for the Firebase
// ref.
func (r *Ref) clientAndRequest(method string, body io.Reader, opts ...QueryOption) (*http.Client, *http.Request, error) {
	var err error

	// get client
	client, err := r.httpClient()
	if err != nil {
		return nil, nil, &Error{
			Err: fmt.Sprintf("could not create client: %v", err),
		}
	}

	// create request
	req, err := r.createRequest(method, body, opts...)
	if err != nil {
		return nil, nil, &Error{
			Err: fmt.Sprintf("could not create request: %v", err),
		}
	}

	return client, req, nil
}

// AddTokenSourceClaim adds a claim to the auth token source.
func (r *Ref) AddTokenSourceClaim(field string, v interface{}) error {
	r.rw.Lock()
	defer r.rw.Unlock()

	if r.auth == nil {
		return &Error{
			Err: "ref does not have an initialized auth token source",
		}
	}

	r.auth.AddClaim(field, v)
	r.source = oauth2.ReuseTokenSource(nil, r.auth)

	return nil
}

// SetQueryOptions sets the default query options for the Firebase ref.
func (r *Ref) SetQueryOptions(opts ...QueryOption) {
	r.rw.Lock()
	defer r.rw.Unlock()

	r.queryOpts = opts
}

// Ref creates a child ref, locking it to the specified path.
//
// If an Option is passed that returns an error, then this func will panic.
//
// If an Option that could return an error needs to be applied to the child ref
// after it has been created, then apply it in the following manner:
//
//     child := db.Ref("/path/to/child")
//     err := SomeOption(child)
// 	   if err != nil { ... }
func (r *Ref) Ref(path string, opts ...Option) *Ref {
	r.rw.RLock()
	defer r.rw.RUnlock()

	curpath := r.url.Path
	if !strings.HasSuffix(curpath, "/") {
		curpath += "/"
	}

	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	c := &Ref{
		url: &url.URL{
			Scheme: r.url.Scheme,
			Opaque: r.url.Opaque,
			User:   r.url.User,
			Host:   r.url.Host,
			Path:   curpath + path,
		},
		transport:   r.transport,
		source:      r.source,
		queryOpts:   r.queryOpts,
		watchBufLen: r.watchBufLen,
	}

	// apply opts
	for _, o := range opts {
		err := o(c)
		if err != nil {
			// options that could error out should not be applied here
			panic(err)
		}
	}

	return c
}

// URL returns the URL for the Firebase ref.
func (r *Ref) URL() *url.URL {
	return r.url
}

// Get retrieves the values stored at the Firebase ref and decodes them into d.
func (r *Ref) Get(d interface{}, opts ...QueryOption) error {
	return Get(r, d, opts...)
}

// Set stores values v at the Firebase ref.
func (r *Ref) Set(v interface{}) error {
	return Set(r, v)
}

// Push pushes values v to the Firebase ref, returning the name (ID) of the
// pushed node.
func (r *Ref) Push(v interface{}) (string, error) {
	return Push(r, v)
}

// Update updates the values stored at the Firebase ref to v.
func (r *Ref) Update(v interface{}) error {
	return Update(r, v)
}

// Remove removes the values stored at the Firebase ref.
func (r *Ref) Remove() error {
	return Remove(r)
}

// SetRules sets the security rules for the Firebase ref.
func (r *Ref) SetRules(v interface{}) error {
	return SetRules(r, v)
}

// SetRulesJSON sets the JSON-encoded security rules for the Firebase ref.
func (r *Ref) SetRulesJSON(buf []byte) error {
	return SetRulesJSON(r, buf)
}

// GetRulesJSON retrieves the security rules for the Firebase ref.
func (r *Ref) GetRulesJSON() ([]byte, error) {
	return GetRulesJSON(r)
}

// Watch watches the Firebase ref for events, emitting encountered events on
// the returned channel. Watch ends when the passed context is done, when the
// remote connection is closed, or when an error is encountered while reading
// events from the server.
//
// NOTE: the Log option will not work with Watch/Listen.
func (r *Ref) Watch(ctxt context.Context, opts ...QueryOption) (<-chan *Event, error) {
	return Watch(r, ctxt, opts...)
}

// Listen listens on the Firebase ref for any of the the specified eventTypes,
// emitting them on the returned channel.
//
// The returned channel is closed only when the context is done. If the
// Firebase connection closes, or the auth token is revoked, then Listen will
// continue to reattempt connecting to the Firebase ref.
//
// NOTE: the Log option will not work with Watch/Listen.
// events from the server.
func (r *Ref) Listen(ctxt context.Context, eventTypes []EventType, opts ...QueryOption) <-chan *Event {
	return Listen(r, ctxt, eventTypes, opts...)
}
