package firebase

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/context"
)

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
func Watch(r *DatabaseRef, ctxt context.Context, opts ...QueryOption) (<-chan *Event, error) {
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
func Listen(r *DatabaseRef, ctxt context.Context, eventTypes []EventType, opts ...QueryOption) <-chan *Event {
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
