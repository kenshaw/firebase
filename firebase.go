package firebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DoRequest does a request against Firebase ref r with the supplied values v,
// decoding the response to d.
func DoRequest(method string, r *Ref, v, d interface{}, opts ...QueryOption) error {
	var err error

	// encode v
	var body io.Reader
	if v != nil {
		buf, err := json.Marshal(v)
		if err != nil {
			return &Error{
				Err: fmt.Sprintf("could not marshal json: %v", err),
			}
		}
		body = bytes.NewReader(buf)
	}

	// create client and request
	client, req, err := r.clientAndRequest(method, body, opts...)
	if err != nil {
		return err
	}

	// do request
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

// Get retrieves the values stored at Firebase reference r and decodes them into d.
func Get(r *Ref, d interface{}, opts ...QueryOption) error {
	return DoRequest("GET", r, nil, d, opts...)
}

// Set stores values v at Firebase reference r.
func Set(r *Ref, v interface{}) error {
	return DoRequest("PUT", r, v, nil)
}

// Push pushes values v to Firebase reference r, returning the name (ID) of the
// created ref.
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
