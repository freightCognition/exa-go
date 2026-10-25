package exa

import (
	"strings"
	"testing"
)

func readAll(t *testing.T, in string) []sseEvent {
	t.Helper()
	r := newSSEReader(strings.NewReader(in))
	var out []sseEvent
	for {
		ev, err := r.next()
		if err != nil {
			if err.Error() != "EOF" {
				t.Fatalf("next: %v", err)
			}
			return out
		}
		out = append(out, ev)
	}
}

func TestSSEBasic(t *testing.T) {
	evs := readAll(t, "event: delta\ndata: {\"a\":1}\n\ndata: {\"b\":2}\n\n")
	if len(evs) != 2 {
		t.Fatalf("got %d events", len(evs))
	}
	if evs[0].Event != "delta" || evs[0].Data != `{"a":1}` {
		t.Errorf("ev0 = %+v", evs[0])
	}
	if evs[1].Event != "" || evs[1].Data != `{"b":2}` {
		t.Errorf("ev1 = %+v", evs[1])
	}
}

func TestSSEMultilineDataCommentsAndCRLF(t *testing.T) {
	evs := readAll(t, ": keepalive\r\ndata: line1\r\ndata: line2\r\nid: 7\r\n\r\n")
	if len(evs) != 1 || evs[0].Data != "line1\nline2" || evs[0].ID != "7" {
		t.Fatalf("evs = %+v", evs)
	}
}

func TestSSENoSpaceAfterColonAndTrailingEventWithoutBlankLine(t *testing.T) {
	evs := readAll(t, "data:x\n\ndata: y")
	if len(evs) != 2 || evs[0].Data != "x" || evs[1].Data != "y" {
		t.Fatalf("evs = %+v", evs)
	}
}

func TestSSESkipsEmptyEvents(t *testing.T) {
	evs := readAll(t, "\n\n: only a comment\n\n")
	if len(evs) != 0 {
		t.Fatalf("evs = %+v", evs)
	}
}

func TestSSELongLine(t *testing.T) {
	long := strings.Repeat("x", 3<<20)
	evs := readAll(t, "data: "+long+"\n\n")
	if len(evs) != 1 || len(evs[0].Data) != len(long) {
		t.Fatalf("long line lost")
	}
}
