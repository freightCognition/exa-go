package exa

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"
)

// sseEvent is one server-sent event.
type sseEvent struct {
	Event string
	Data  string
	ID    string
}

type sseReader struct{ r *bufio.Reader }

func newSSEReader(r io.Reader) *sseReader { return &sseReader{r: bufio.NewReader(r)} }

// next returns the next event with data, or io.EOF.
func (s *sseReader) next() (sseEvent, error) {
	var ev sseEvent
	var data []string
	hasData := false
	for {
		line, err := s.r.ReadString('\n')
		if err != nil && err != io.EOF {
			return sseEvent{}, err
		}
		eof := err == io.EOF
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if hasData {
				ev.Data = strings.Join(data, "\n")
				return ev, nil
			}
			if eof {
				return sseEvent{}, io.EOF
			}
			ev = sseEvent{}
			continue
		}
		if !strings.HasPrefix(line, ":") {
			field, value, _ := strings.Cut(line, ":")
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				ev.Event = value
			case "data":
				data = append(data, value)
				hasData = true
			case "id":
				ev.ID = value
			}
		}
		if eof {
			if hasData {
				ev.Data = strings.Join(data, "\n")
				return ev, nil
			}
			return sseEvent{}, io.EOF
		}
	}
}

// stream sends a request asking for server-sent events and yields each event
// decoded into T. decode may return (zero, false, nil) to skip an event, and a
// non-nil error to end the stream with that error. A "[DONE]" data line ends
// the stream. The response is closed when the caller stops ranging.
func stream[T any](ctx context.Context, c *Client, method, base, path string, query url.Values, body any, decode func(sseEvent) (T, bool, error), edits ...func(*http.Request)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		req, err := c.newRequest(ctx, method, base, path, query, body)
		if err != nil {
			yield(zero, err)
			return
		}
		req.Header.Set("Accept", "text/event-stream")
		for _, edit := range edits {
			edit(req)
		}
		resp, err := c.send(req)
		if err != nil {
			yield(zero, err)
			return
		}
		defer resp.Body.Close()
		r := newSSEReader(resp.Body)
		for {
			ev, err := r.next()
			if err == io.EOF {
				return
			}
			if err != nil {
				if ctx.Err() != nil {
					err = ctx.Err()
				}
				yield(zero, fmt.Errorf("exa: read %s stream: %w", path, err))
				return
			}
			if ev.Data == "[DONE]" {
				return
			}
			v, ok, err := decode(ev)
			if err != nil {
				yield(zero, err)
				return
			}
			if ok && !yield(v, nil) {
				return
			}
		}
	}
}

// decodeJSONEvent is the default decode: the event data is one JSON T.
func decodeJSONEvent[T any](ev sseEvent) (T, bool, error) {
	var v T
	if err := json.Unmarshal([]byte(ev.Data), &v); err != nil {
		return v, false, fmt.Errorf("exa: decode stream event: %w", err)
	}
	return v, true, nil
}
