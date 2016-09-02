package firebase

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
)

// EventType is a Firebase event type.
type EventType string

const (
	// firebase SSE events
	EventTypePut         EventType = "put"
	EventTypePatch       EventType = "patch"
	EventTypeKeepAlive   EventType = "keep-alive"
	EventTypeCancel      EventType = "cancel"
	EventTypeAuthRevoked EventType = "auth_revoked"

	// synthesized events
	EventTypeClosed              EventType = "closed"
	EventTypeUnknownError        EventType = "unknown_error"
	EventTypeMalformedEventError EventType = "malformed_event_error"
	EventTypeMalformedDataError  EventType = "malformed_data_error"
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

// Watch watches a Firebase ref for events, emitting them on returned channel.
// Will end when the passed context is canceled or when the remote connection
// is closed.
func Watch(r *Ref, ctxt context.Context, opts ...QueryOption) (<-chan *Event, error) {
	// get client
	client, err := r.httpClient()
	if err != nil {
		return nil, fmt.Errorf("firebase: could not create client: %v", err)
	}

	// create request
	req, err := r.createRequest("GET", nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase: could not create request: %v", err)
	}

	// set headers
	req.Header.Add("Accept", "text/event-stream")

	// execute
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	events := make(chan *Event, r.watchBufLen)
	go func() {
		defer res.Body.Close()

		// create reader
		rdr := bufio.NewReader(res.Body)

		for {
			select {
			default:
				// read event: line
				line, err := rdr.ReadBytes('\n')
				if err == io.EOF {
					events <- &Event{Type: EventTypeClosed}
					close(events)
					return
				} else if err != nil {
					events <- &Event{Type: EventTypeUnknownError, Data: []byte(err.Error())}
					close(events)
					return
				}

				// check line has event prefix
				if !bytes.HasPrefix(line, []byte(watchEventPrefix)) {
					events <- &Event{Type: EventTypeMalformedEventError}
					close(events)
					return
				}

				// read data: line
				data, err := rdr.ReadBytes('\n')
				if err == io.EOF {
					events <- &Event{Type: EventTypeClosed}
					close(events)
					return
				} else if err != nil {
					events <- &Event{Type: EventTypeUnknownError, Data: []byte(err.Error())}
					close(events)
					return
				}

				// check data has event prefix
				if !bytes.HasPrefix(data, []byte(watchDataPrefix)) {
					events <- &Event{Type: EventTypeMalformedDataError}
					close(events)
					return
				}

				// emit event
				events <- &Event{
					Type: EventType(bytes.TrimSpace(line[len(watchEventPrefix):])),
					Data: bytes.TrimSpace(data[len(watchDataPrefix):]),
				}

				// consume empty line
				empty, err := rdr.ReadBytes('\n')
				if err == io.EOF {
					events <- &Event{Type: EventTypeClosed}
					close(events)
					return
				} else if err != nil {
					events <- &Event{Type: EventTypeUnknownError, Data: []byte(err.Error())}
					close(events)
					return
				}
				empty = bytes.TrimSpace(empty)
				if len(empty) > 0 {
					events <- &Event{Type: EventTypeUnknownError, Data: []byte(fmt.Sprintf("expected empty line, got: %s", string(empty)))}
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
