// Package firebase provides Firebase v3+ compatible clients.
package firebase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

const (
	// DefaultWatchBuffer is the default length of an event channel created on
	// a call to Watch.
	DefaultWatchBuffer = 64
)

// OpType is the Firebase operation type.
type OpType string

const (
	// OpTypeGet is the Firebase Push operation.
	OpTypeGet OpType = "GET"

	// OpTypePush is the Firebase Push operation.
	OpTypePush OpType = "POST"

	// OpTypeSet is the Firebase Set operation.
	OpTypeSet OpType = "PUT"

	// OpTypeUpdate is the Firebase Update operation.
	OpTypeUpdate OpType = "PATCH"

	// OpTypeRemove is the Firebase Remove operation.
	OpTypeRemove OpType = "DELETE"
)

// Do executes an HTTP operation on Firebase database ref r passing the
// supplied value v as JSON marshaled data and decoding the response to d.
func Do(op OpType, r *DatabaseRef, v, d interface{}, opts ...QueryOption) error {
	var err error

	// encode v
	var body io.Reader
	switch x := v.(type) {
	case io.Reader:
		body = x

	case []byte:
		body = bytes.NewReader(x)

	default:
		if v != nil {
			buf, err := json.Marshal(v)
			if err != nil {
				return &Error{
					Err: fmt.Sprintf("could not marshal json: %v", err),
				}
			}
			body = bytes.NewReader(buf)
		}
	}

	// create client and request
	client, req, err := r.clientAndRequest(string(op), body, opts...)
	if err != nil {
		return err
	}

	// execute
	res, err := client.Do(req)
	if err != nil {
		return &Error{
			Err: fmt.Sprintf("could not execute request: %v", err),
		}
	}
	defer res.Body.Close()

	// check for server error
	err = checkServerError(res)
	if err != nil {
		return err
	}

	// decode body to d
	if d != nil {
		dec := json.NewDecoder(res.Body)
		dec.UseNumber()
		err = dec.Decode(d)
		if err != nil {
			return &Error{
				Err: fmt.Sprintf("could not unmarshal json: %v", err),
			}
		}
	}

	return nil
}

// Get retrieves the values stored at Firebase database ref r and decodes them
// into d.
func Get(r *DatabaseRef, d interface{}, opts ...QueryOption) error {
	return Do(OpTypeGet, r, nil, d, opts...)
}

// Set stores values v at Firebase database ref r.
func Set(r *DatabaseRef, v interface{}, opts ...QueryOption) error {
	return Do(OpTypeSet, r, v, nil, opts...)
}

// Push pushes values v to Firebase database ref r, returning the name (ID) of
// the pushed node.
func Push(r *DatabaseRef, v interface{}, opts ...QueryOption) (string, error) {
	var res struct {
		Name string `json:"name"`
	}

	err := Do(OpTypePush, r, v, &res, opts...)
	if err != nil {
		return "", err
	}

	return res.Name, nil
}

// Update updates the values stored at Firebase database ref r to v.
func Update(r *DatabaseRef, v interface{}, opts ...QueryOption) error {
	return Do(OpTypeUpdate, r, v, nil, opts...)
}

// Remove removes the values stored at Firebase database ref r.
func Remove(r *DatabaseRef, opts ...QueryOption) error {
	return Do(OpTypeRemove, r, nil, nil, opts...)
}

// SetRules sets the security rules for Firebase database ref r.
func SetRules(r *DatabaseRef, v interface{}) error {
	return Do(OpTypeSet, r.Ref("/.settings/rules"), v, nil)
}

// SetRulesJSON sets the JSON-encoded security rules for Firebase database ref
// r.
func SetRulesJSON(r *DatabaseRef, buf []byte) error {
	var err error
	var v interface{}

	// decode
	d := json.NewDecoder(bytes.NewReader(buf))
	d.UseNumber()
	err = d.Decode(&v)
	if err != nil {
		return &Error{
			Err: fmt.Sprintf("could not decode json: %v", err),
		}
	}

	// encode
	var rules bytes.Buffer
	e := json.NewEncoder(&rules)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err = e.Encode(&v)
	if err != nil {
		return &Error{
			Err: fmt.Sprintf("could not encode json: %v", err),
		}
	}

	return Do(OpTypeSet, r.Ref("/.settings/rules"), rules.Bytes(), nil)
}

// GetRulesJSON retrieves the security rules for Firebase database ref r.
func GetRulesJSON(r *DatabaseRef) ([]byte, error) {
	var d json.RawMessage
	err := Do(OpTypeSet, r.Ref("/.settings/rules"), nil, &d)
	if err != nil {
		return nil, err
	}
	return []byte(d), nil
}

// DatabaseRef is a Firebase database reference.
type DatabaseRef struct {
	rw sync.RWMutex

	url       *url.URL
	transport http.RoundTripper

	// source is the oauth2 token source.
	source oauth2.TokenSource

	queryOpts []QueryOption

	watchBufLen int
}

// NewDatabaseRef creates a new Firebase base database ref using the supplied
// options.
func NewDatabaseRef(opts ...Option) (*DatabaseRef, error) {
	var err error

	// create client
	r := &DatabaseRef{
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
func (r *DatabaseRef) httpClient() (*http.Client, error) {
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

// createRequest creates a http.Request for the Firebase database ref with
// method, body, and query opts.
func (r *DatabaseRef) createRequest(method string, body io.Reader, opts ...QueryOption) (*http.Request, error) {
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
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}

	// substitute + on raw path
	if strings.Contains(req.URL.Path, "+") {
		req.URL.RawPath = strings.Replace(req.URL.Path, "+", "%2B", -1)
	}

	return req, nil
}

// clientAndRequest creates a *http.Client and *http.Request for the Firebase
// ref.
func (r *DatabaseRef) clientAndRequest(method string, body io.Reader, opts ...QueryOption) (*http.Client, *http.Request, error) {
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

// Ref creates a new Firebase database child ref, locked to the specified path.
//
// NOTE: any Option passed returning an error will cause this func to panic.
// Instead if an Option might return an error, then it should be applied after
// the child ref has been created in the following manner:
//
//     child := db.Ref("/path/to/child")
//     err := SomeOption(child)
// 	   if err != nil { log.Fatal(err) }
func (r *DatabaseRef) Ref(path string, opts ...Option) *DatabaseRef {
	r.rw.RLock()
	defer r.rw.RUnlock()

	// create new path
	curpath := r.url.Path
	if !strings.HasSuffix(curpath, "/") {
		curpath += "/"
	}
	path = strings.TrimPrefix(path, "/")

	// create child ref
	c := &DatabaseRef{
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

// URL returns the URL for the Firebase database ref.
func (r *DatabaseRef) URL() *url.URL {
	return r.url
}

// Get retrieves the values stored at the Firebase database ref and decodes
// them into d.
func (r *DatabaseRef) Get(d interface{}, opts ...QueryOption) error {
	return Get(r, d, opts...)
}

// Set stores values v at the Firebase database ref.
func (r *DatabaseRef) Set(v interface{}, opts ...QueryOption) error {
	return Set(r, v, opts...)
}

// Push pushes values v to the Firebase database ref, returning the name (ID)
// of the pushed node.
func (r *DatabaseRef) Push(v interface{}, opts ...QueryOption) (string, error) {
	return Push(r, v, opts...)
}

// Update updates the values stored at the Firebase database ref to v.
func (r *DatabaseRef) Update(v interface{}, opts ...QueryOption) error {
	return Update(r, v, opts...)
}

// Remove removes the values stored at the Firebase database ref.
func (r *DatabaseRef) Remove(opts ...QueryOption) error {
	return Remove(r, opts...)
}

// SetRules sets the security rules for the Firebase database ref.
func (r *DatabaseRef) SetRules(v interface{}) error {
	return SetRules(r, v)
}

// SetRulesJSON sets the JSON-encoded security rules for the Firebase database
// ref.
func (r *DatabaseRef) SetRulesJSON(buf []byte) error {
	return SetRulesJSON(r, buf)
}

// GetRulesJSON retrieves the security rules for the Firebase database ref.
func (r *DatabaseRef) GetRulesJSON() ([]byte, error) {
	return GetRulesJSON(r)
}

// Watch watches the Firebase database ref for events, emitting encountered
// events on the returned channel. Watch ends when the passed context is done,
// when the remote connection is closed, or when an error is encountered while
// reading events from the server.
//
// NOTE: the Log option will not work with Watch/Listen.
func (r *DatabaseRef) Watch(ctxt context.Context, opts ...QueryOption) (<-chan *Event, error) {
	return Watch(r, ctxt, opts...)
}

// Listen listens on the Firebase database ref for any of the the specified
// eventTypes, emitting them on the returned channel.
//
// The returned channel is closed only when the context is done. If the
// Firebase connection closes, or the auth token is revoked, then Listen will
// continue to reattempt connecting to the Firebase database ref.
//
// NOTE: the Log option will not work with Watch/Listen.
func (r *DatabaseRef) Listen(ctxt context.Context, eventTypes []EventType, opts ...QueryOption) <-chan *Event {
	return Listen(r, ctxt, eventTypes, opts...)
}
