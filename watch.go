package firebase

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/context"
)

// EventType is a Firebase event type.
type EventType string

const (
	// firebase SSE events

	// EventTypePut is the event type sent when new data is inserted to the
	// Firebase ref.
	EventTypePut EventType = "put"

	// EventTypePatch is the event type sent when data at the Firebase ref is
	// updated.
	EventTypePatch EventType = "patch"

	// EventTypeKeepAlive is the event type sent when a keep alive is
	// encountered.
	EventTypeKeepAlive EventType = "keep-alive"

	// EventTypeCancel is the event type sent when the Firebase security rules
	// on the watched ref are altered to no longer allow the auth token to read
	// it.
	EventTypeCancel EventType = "cancel"

	// EventTypeAuthRevoked is the event type sent when the auth token is
	// revoked or expired.
	EventTypeAuthRevoked EventType = "auth_revoked"

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

// Event is a Firebase event emitted from Watch.
type Event struct {
	Type EventType
	Data []byte
}

// String satisfies the stringer interface.
func (e Event) String() string {
	return fmt.Sprintf("%s: %s", e.Type, string(e.Data))
}

const (
	watchEventPrefix = "event: "
	watchDataPrefix  = "data: "
)

// readLine reads a line from a reader, synthesizing the provided event type if
// there was an error or the line is missing the supplied prefix.
func readLine(rdr *bufio.Reader, prefix string, errEventType EventType) ([]byte, *Event) {
	// read event: line
	line, err := rdr.ReadBytes('\n')
	if err == io.EOF {
		return nil, &Event{
			Type: EventTypeClosed,
			Data: []byte("connection closed"),
		}
	} else if err != nil {
		return nil, &Event{
			Type: EventTypeUnknownError,
			Data: []byte(err.Error()),
		}
	}

	// empty line check for empty prefix
	if len(prefix) == 0 {
		line = bytes.TrimSpace(line)
		if len(line) != 0 {
			return nil, &Event{
				Type: errEventType,
				Data: []byte("expected empty line"),
			}
		}
		return line, nil
	}

	// check line has event prefix
	if !bytes.HasPrefix(line, []byte(prefix)) {
		return nil, &Event{
			Type: errEventType,
			Data: []byte("missing prefix"),
		}
	}

	// trim space
	return bytes.TrimSpace(line[len([]byte(prefix)):]), nil
}

// Watch watches a Firebase ref for events, emitting them on returned channel.
// Will end when the passed context is canceled or when the remote connection
// is closed.
func Watch(r *Ref, ctxt context.Context, opts ...QueryOption) (<-chan *Event, error) {
	var err error

	// get client and request
	client, req, err := r.clientAndRequest("GET", nil, opts...)
	if err != nil {
		return nil, err
	}

	// set request headers
	req.Header.Add("Accept", "text/event-stream")

	// execute
	res, err := client.Do(req)
	if err != nil {
		return nil, &Error{
			Err: fmt.Sprintf("could not execute request: %v", err),
		}
	}

	// check server error
	err = checkServerError(res)
	if err != nil {
		return nil, err
	}

	events := make(chan *Event, r.watchBufLen)
	go func() {
		defer res.Body.Close()

		// create reader
		rdr := bufio.NewReader(res.Body)

		var errEvent *Event
		var typ, data []byte

		for {
			select {
			default:
				// read line "event: <event>"
				typ, errEvent = readLine(rdr, watchEventPrefix, EventTypeMalformedEventError)
				if errEvent != nil {
					events <- errEvent
					close(events)
					return
				}

				// read line "data: <data>"
				data, errEvent = readLine(rdr, watchDataPrefix, EventTypeMalformedDataError)
				if errEvent != nil {
					events <- errEvent
					close(events)
					return
				}

				// emit event
				events <- &Event{
					Type: EventType(typ),
					Data: data,
				}

				// consume empty line
				_, errEvent = readLine(rdr, "", EventTypeUnknownError)
				if errEvent != nil {
					events <- errEvent
					close(events)
					return
				}

			// context finished
			case <-ctxt.Done():
				close(events)
				return
			}
		}
	}()

	return events, nil
}
