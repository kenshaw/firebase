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

const (
	watchEventPrefix = "event: "
	watchDataPrefix  = "data: "
)

// readLine reads a line from a io.Reader, synthesizing errEventType if an
// error was encountered, or the line is missing the supplied prefix.
//
// if the prefix is the empty string, then readLine tests for a blank or empty
// line.
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

// Watch watches a Firebase ref for events, emitting encountered events on the
// returned channel. Watch ends when the passed context is done, when the
// remote connection is closed, or when an error is encountered while reading
// data.
//
// NOTE: the Log option will not work with Watch/Listen.
// events from the server.
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

// Listen listens on a Firebase ref for any of the the specified eventTypes,
// emitting them on the returned channel.
//
// The returned channel is closed only when the context is done. If the
// Firebase connection closes, or the auth token is revoked, then Listen will
// continue to reattempt connecting to the Firebase ref.
//
// NOTE: the Log option will not work with Watch/Listen.
// events from the server.
func Listen(r *Ref, ctxt context.Context, eventTypes []EventType, opts ...QueryOption) <-chan *Event {
	events := make(chan *Event, r.watchBufLen)

	go func() {
		for {
		watchLoop:
			select {
			default:
				// setup watch
				ev, err := Watch(r, ctxt, opts...)
				if err != nil {
					close(events)
					return
				}

				// consume events
				for e := range ev {
					if e == nil {
						break watchLoop
					}

					// filter
					for _, typ := range eventTypes {
						if typ == e.Type {
							events <- e
						}
					}
				}

			case <-ctxt.Done():
				close(events)
				return
			}
		}
	}()

	return events
}
