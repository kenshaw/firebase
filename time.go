package firebase

import (
	"strconv"
	"time"
)

const (
	// serverTimestampValue is the special value sent to Firebase for server
	// timestamps.
	serverTimestampValue = `{".sv":"timestamp"}`
)

// ServerTimestamp provides a json.Marshal'able (and Unmarshal'able) type for
// use with Firebase.
//
// When this type has a zero value, and is serialized to Firebase, Firebase
// will store the current time in milliseconds since the Unix epoch. When the
// value is unserialized from Firebase, then the stored time (ie, milliseconds
// since the Unix epoch) will be returned.
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
// The Firebase representation of time is a JSON Number of milliseconds since
// the Unix epoch.
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

// Error satisfies the error interface.
func (e *Error) Error() string {
	return "firebase: " + e.Err
}
