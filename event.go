package firebase

import "fmt"

// EventType is a Firebase event type.
type EventType string

const (
	// -----------------------------------------------
	// firebase server-sent events

	// EventTypePut is the event type sent when new data is inserted to a
	// watched Firebase ref.
	EventTypePut EventType = "put"

	// EventTypePatch is the event type sent when data is updated at a watched
	// Firebase ref.
	EventTypePatch EventType = "patch"

	// EventTypeKeepAlive is the event type sent when a keep alive is
	// encountered.
	EventTypeKeepAlive EventType = "keep-alive"

	// EventTypeCancel is the event type sent when the Firebase security rules
	// on the watched ref are altered to no longer allow the auth token to read
	// data at the watched ref.
	EventTypeCancel EventType = "cancel"

	// EventTypeAuthRevoked is the event type sent when the auth token is
	// revoked or expired.
	EventTypeAuthRevoked EventType = "auth_revoked"

	// -----------------------------------------------
	// synthesized events

	// EventTypeClosed is the event type sent when the connection with the
	// Firebase server is closed.
	EventTypeClosed EventType = "closed"

	// EventTypeUnknownError is the event type sent when an unknown error is
	// encountered.
	EventTypeUnknownError EventType = "unknown_error"

	// EventTypeMalformedEventError is the event type sent when a malformed
	// event is read from the Firebase server.
	EventTypeMalformedEventError EventType = "malformed_event_error"

	// EventTypeMalformedDataError is the event type sent when malformed data
	// is read from the Firebase server.
	EventTypeMalformedDataError EventType = "malformed_data_error"
)

// String satisfies the stringer interface.
func (e EventType) String() string {
	return string(e)
}

// Event is a Firebase server side event emitted from Watch and Listen.
type Event struct {
	Type EventType
	Data []byte
}

// String satisfies the stringer interface.
func (e Event) String() string {
	return fmt.Sprintf("%s: %s", e.Type, string(e.Data))
}
