// Package firebase provides a Firebase v3.0.0+ compatible API implementation.
package firebase

import (
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
			return nil, &Error{
				Err: fmt.Sprintf("could not create database ref: %v", err),
			}
		}
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

// AddClaim adds a claim to the Firebase ref token source.
func (r *Ref) AddClaim(field string, v interface{}) error {
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

// Ref duplicates the ref, but locking it to a sub Firebase ref at path.
func (r *Ref) Ref(path string, opts ...Option) *Ref {
	r.rw.RLock()
	defer r.rw.RUnlock()

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if strings.HasSuffix(r.url.Path, "/") {
		path = r.url.Path[:len(r.url.Path)-1] + path
	}

	c := &Ref{
		url: &url.URL{
			Scheme: r.url.Scheme,
			Opaque: r.url.Opaque,
			User:   r.url.User,
			Host:   r.url.Host,
			Path:   path,
		},
		transport:   r.transport,
		source:      r.source,
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

// Get retrieves the values at the Firebase ref, decoding to d.
func (r *Ref) Get(d interface{}, opts ...QueryOption) error {
	return Get(r, d, opts...)
}

// Set stores values v at Firebase reference r.
func (r *Ref) Set(v interface{}) error {
	return Set(r, v)
}

// Push pushes values v to Firebase reference r, returning the ID.
func (r *Ref) Push(v interface{}) (string, error) {
	return Push(r, v)
}

// Update updates the stored values at Firebase reference r to v.
func (r *Ref) Update(v interface{}) error {
	return Update(r, v)
}

// Remove removes the values stored at Firebase reference r.
func (r *Ref) Remove() error {
	return Remove(r)
}

// GetRules retrieves the security rules for Firebase reference r.
func (r *Ref) GetRules() ([]byte, error) {
	return GetRules(r)
}

// SetRules sets the security rules for Firebase reference r.
func (r *Ref) SetRules(v interface{}) error {
	return SetRules(r, v)
}

// Watch watches a Firebase ref for events, emitting them on returned channel.
// Will end when the passed context is done.
func (r *Ref) Watch(ctxt context.Context) (<-chan *Event, error) {
	return Watch(r, ctxt)
}
