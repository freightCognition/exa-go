package exa

import (
	"context"
	"errors"
	"testing"
)

func TestListParamsValues(t *testing.T) {
	if v := (*ListParams)(nil).values(); len(v) != 0 {
		t.Errorf("nil params = %v", v)
	}
	v := (&ListParams{Cursor: "c1", Limit: 10}).values()
	if v.Get("cursor") != "c1" || v.Get("limit") != "10" {
		t.Errorf("values = %v", v)
	}
	if v := (&ListParams{}).values(); len(v) != 0 {
		t.Errorf("zero params = %v", v)
	}
}

func TestAllWalksEveryPage(t *testing.T) {
	pages := map[string]*Page[int]{
		"":   {Data: []int{1, 2}, HasMore: true, NextCursor: Ptr("p2")},
		"p2": {Data: []int{3}, HasMore: true, NextCursor: Ptr("p3")},
		"p3": {Data: []int{4}, HasMore: false},
	}
	var cursors []string
	var got []int
	for v, err := range All(context.Background(), func(_ context.Context, cursor string) (*Page[int], error) {
		cursors = append(cursors, cursor)
		return pages[cursor], nil
	}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if len(got) != 4 || got[3] != 4 || len(cursors) != 3 {
		t.Fatalf("got %v via cursors %v", got, cursors)
	}
}

func TestAllStopsEarlyAndOnError(t *testing.T) {
	calls := 0
	for range All(context.Background(), func(context.Context, string) (*Page[int], error) {
		calls++
		return &Page[int]{Data: []int{1, 2, 3}, HasMore: true, NextCursor: Ptr("more")}, nil
	}) {
		break
	}
	if calls != 1 {
		t.Errorf("fetched %d pages after break", calls)
	}

	boom := errors.New("boom")
	var gotErr error
	for _, err := range All(context.Background(), func(context.Context, string) (*Page[int], error) {
		return nil, boom
	}) {
		gotErr = err
	}
	if !errors.Is(gotErr, boom) {
		t.Errorf("err = %v", gotErr)
	}
}

func TestAllStopsWhenCursorMissing(t *testing.T) {
	calls := 0
	for range All(context.Background(), func(context.Context, string) (*Page[int], error) {
		calls++
		return &Page[int]{Data: []int{1}, HasMore: true}, nil // hasMore without a cursor
	}) {
	}
	if calls != 1 {
		t.Errorf("looped %d times on a page with no next cursor", calls)
	}
}
