package firebase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	serverTimestampValue = `{".sv":"timestamp"}`
)

// ServerTimestamp provides a json.Marshal'able (and Unmarshal'able) type for
// use with Firebase.
type ServerTimestamp time.Time

// MarshalJSON satisfies the json.Marshaler interface.
func (st ServerTimestamp) MarshalJSON() ([]byte, error) {
	t := time.Time(st)

	// special firebase value
	if t.IsZero() {
		return []byte(serverTimestampValue), nil
	}

	return []byte(strconv.FormatInt(t.UnixNano()/int64(time.Millisecond), 10)), nil
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (st *ServerTimestamp) UnmarshalJSON(buf []byte) error {
	// special firebase value
	v := string(buf)
	switch v {
	case serverTimestampValue:
		*st = ServerTimestamp(time.Now())
		return nil

	case "null":
		*st = ServerTimestamp{}
		return nil
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return err
	}

	*st = ServerTimestamp(time.Unix(0, i*int64(time.Millisecond)))
	return nil
}

// Time returns the ServerTimestamp as time.Time.
func (st ServerTimestamp) Time() time.Time {
	return time.Time(st)
}

// String satisfies the stringer interface.
func (st ServerTimestamp) String() string {
	return time.Time(st).String()
}

// Time provides a json.Marshal'able (and Unmarshal'able) type for that is
// compatible with Firebase server timestamps.
//
// The JSON representation of time is a JSON number of milliseconds since the
// Unix epoch.
type Time time.Time

// MarshalJSON satisfies the json.Marshaler interface.
func (t Time) MarshalJSON() ([]byte, error) {
	z := time.Time(t)

	return []byte(strconv.FormatInt(z.UnixNano()/int64(time.Millisecond), 10)), nil
}

// UnmarshalJSON satisfies the json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(buf []byte) error {
	v := string(buf)
	if v == "null" {
		*t = Time{}
		return nil
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return err
	}

	*t = Time(time.Unix(0, i*int64(time.Millisecond)))
	return nil
}

// Time returns the Time as time.Time.
func (t Time) Time() time.Time {
	return time.Time(t)
}

// String satisfies the stringer interface.
func (t Time) String() string {
	return time.Time(t).String()
}

// Error is a general Firebase error.
type Error struct {
	Err string `json:"error"`
}

// Error satisifies the error interface.
func (e *Error) Error() string {
	return "firebase: " + e.Err
}

// checkServerError looks at a http.Response and determines if it encountered
// an error, and marshals the error into a Error if it did.
func checkServerError(res *http.Response) error {
	// some kind of server error
	if res.StatusCode < 200 || res.StatusCode > 299 {
		buf, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return &Error{
				Err: fmt.Sprintf("unable to read server error: %v", err),
			}
		}
		if len(buf) < 1 {
			return &Error{
				Err: fmt.Sprintf("empty server error: %s (%d)", res.Status, res.StatusCode),
			}
		}

		var e Error
		err = json.Unmarshal(buf, &e)
		if err != nil {
			return &Error{
				Err: fmt.Sprintf("unknown server error: %s (%d)", string(buf), res.StatusCode),
			}
		}

		return &e
	}

	return nil
}
