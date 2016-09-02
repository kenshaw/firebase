package firebase

import (
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
	if string(buf) == serverTimestampValue {
		*st = ServerTimestamp(time.Now())
		return nil
	}

	i, err := strconv.ParseInt(string(buf), 10, 64)
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

// ServerError is a Firebase server error.
type ServerError struct {
	Err string `json:"error"`
}

// Error satisifies the error interface.
func (se ServerError) Error() string {
	return "firebase: got server error: " + se.Err
}
