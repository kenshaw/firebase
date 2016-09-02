package firebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// DoRawRequest performs a raw request against Firebase ref r, with the
// supplied body and request options.
func DoRawRequest(method string, r *Ref, body io.Reader, opts ...QueryOption) (*http.Response, error) {
	var err error

	// get client
	client, err := r.httpClient()
	if err != nil {
		return nil, fmt.Errorf("firebase: could not create client: %v", err)
	}

	// create request
	req, err := r.createRequest(method, body, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase: could not create request: %v", err)
	}

	return client.Do(req)
}

// DoRequest does a request against Firebase ref r with the supplied values v,
// decoding the response to d.
func DoRequest(method string, r *Ref, v, d interface{}, opts ...QueryOption) error {
	var err error

	// encode v
	var body io.Reader
	if v != nil {
		buf, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("firebase: could not marshal json: %v", err)
		}
		body = bytes.NewReader(buf)
	}

	// do request
	res, err := DoRawRequest(method, r, body, opts...)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// some kind of server error
	if res.StatusCode < 200 || res.StatusCode > 299 {
		buf, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("firebase: could not read server error: %v", err)
		}

		var se ServerError
		err = json.Unmarshal(buf, &se)
		if err != nil && len(buf) > 0 {
			return &ServerError{
				Err: fmt.Sprintf("%s (%d)", string(buf), res.StatusCode),
			}
		} else if err != nil {
			return fmt.Errorf("firebase: could not decode server error: %v", err)
		}

		return &se
	}

	// decode body to d
	if d != nil {
		dec := json.NewDecoder(res.Body)
		dec.UseNumber()
		err = dec.Decode(d)
		if err != nil {
			return fmt.Errorf("firebase: could not unmarshal json: %v", err)
		}
	}

	return nil
}

// Get retrieves the values stored at Firebase reference r and decodes them into d.
func Get(r *Ref, d interface{}, opts ...QueryOption) error {
	return DoRequest("GET", r, nil, d, opts...)
}

// Set stores values v at Firebase reference r.
func Set(r *Ref, v interface{}) error {
	return DoRequest("PUT", r, v, nil)
}

// Push pushes values v to Firebase reference r, returning the ID.
func Push(r *Ref, v interface{}) (string, error) {
	var res struct {
		Name string `json:"name"`
	}

	err := DoRequest("POST", r, v, &res)
	if err != nil {
		return "", err
	}

	return res.Name, nil
}

// Update updates the stored values at Firebase reference r to v.
func Update(r *Ref, v interface{}) error {
	return DoRequest("PATCH", r, v, nil)
}

// Remove removes the values stored at Firebase reference r.
func Remove(r *Ref) error {
	return DoRequest("DELETE", r, nil, nil)
}

// GetRules retrieves the security rules for Firebase reference r.
func GetRules(r *Ref) ([]byte, error) {
	var d json.RawMessage
	err := DoRequest("GET", r.Ref("/.settings/rules"), nil, &d)
	if err != nil {
		return nil, err
	}
	return []byte(d), nil
}

// SetRules sets the security rules for Firebase reference r.
func SetRules(r *Ref, v interface{}) error {
	return DoRequest("PUT", r.Ref("/.settings/rules"), v, nil)
}
