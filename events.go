package exa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// EventType names a Websets event.
type EventType string

// Websets event types.
const (
	EventWebsetCreated         EventType = "webset.created"
	EventWebsetDeleted         EventType = "webset.deleted"
	EventWebsetPaused          EventType = "webset.paused"
	EventWebsetIdle            EventType = "webset.idle"
	EventWebsetSearchCreated   EventType = "webset.search.created"
	EventWebsetSearchCanceled  EventType = "webset.search.canceled"
	EventWebsetSearchCompleted EventType = "webset.search.completed"
	EventWebsetSearchUpdated   EventType = "webset.search.updated"
	EventImportCreated         EventType = "import.created"
	EventImportCompleted       EventType = "import.completed"
	EventWebsetItemCreated     EventType = "webset.item.created"
	EventWebsetItemEnriched    EventType = "webset.item.enriched"
	EventMonitorCreated        EventType = "monitor.created"
	EventMonitorUpdated        EventType = "monitor.updated"
	EventMonitorDeleted        EventType = "monitor.deleted"
	EventMonitorRunCreated     EventType = "monitor.run.created"
	EventMonitorRunCompleted   EventType = "monitor.run.completed"
	// Export events have no payload schema in the spec; read Event.Data.
	EventWebsetExportCreated   EventType = "webset.export.created"
	EventWebsetExportCompleted EventType = "webset.export.completed"
)

// Event is something that happened in Websets, as listed by Events or
// delivered to a webhook. Data is the affected resource; decode it with the
// accessor matching Type (Webset, Item, Search, Import, Monitor, MonitorRun).
type Event struct {
	ID        string          `json:"id"`
	Object    string          `json:"object"`
	Type      EventType       `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"createdAt"`
}

// eventData maps each event type to the accessor that decodes its Data.
var eventData = map[EventType]string{
	EventWebsetCreated: "Webset", EventWebsetDeleted: "Webset", EventWebsetPaused: "Webset", EventWebsetIdle: "Webset",
	EventWebsetItemCreated: "Item", EventWebsetItemEnriched: "Item",
	EventWebsetSearchCreated: "Search", EventWebsetSearchCanceled: "Search", EventWebsetSearchCompleted: "Search", EventWebsetSearchUpdated: "Search",
	EventImportCreated: "Import", EventImportCompleted: "Import",
	EventMonitorCreated: "Monitor", EventMonitorUpdated: "Monitor", EventMonitorDeleted: "Monitor",
	EventMonitorRunCreated: "MonitorRun", EventMonitorRunCompleted: "MonitorRun",
}

func eventAs[T any](e Event, kind string) (*T, error) {
	if eventData[e.Type] != kind {
		return nil, fmt.Errorf("exa: event %s carries no %s", e.Type, kind)
	}
	var out T
	if err := json.Unmarshal(e.Data, &out); err != nil {
		return nil, fmt.Errorf("exa: decode %s event data: %w", e.Type, err)
	}
	return &out, nil
}

// Webset decodes Data for webset.created, .deleted, .paused and .idle.
func (e Event) Webset() (*Webset, error) { return eventAs[Webset](e, "Webset") }

// Item decodes Data for webset.item.created and .enriched.
func (e Event) Item() (*WebsetItem, error) { return eventAs[WebsetItem](e, "Item") }

// Search decodes Data for webset.search.* events.
func (e Event) Search() (*WebsetSearch, error) { return eventAs[WebsetSearch](e, "Search") }

// Import decodes Data for import.created and .completed.
func (e Event) Import() (*Import, error) { return eventAs[Import](e, "Import") }

// Monitor decodes Data for monitor.created, .updated and .deleted.
func (e Event) Monitor() (*Monitor, error) { return eventAs[Monitor](e, "Monitor") }

// MonitorRun decodes Data for monitor.run.created and .completed.
func (e Event) MonitorRun() (*MonitorRun, error) { return eventAs[MonitorRun](e, "MonitorRun") }

// ListEventsParams filter Events.List.
type ListEventsParams struct {
	ListParams
	Types         []EventType
	CreatedBefore *time.Time
	CreatedAfter  *time.Time
}

// List returns one page of events, newest first.
func (s *WebsetEventsService) List(ctx context.Context, p *ListEventsParams) (*Page[Event], error) {
	var q url.Values
	if p != nil {
		q = p.values()
		for _, t := range p.Types {
			q.Add("types", string(t))
		}
		if p.CreatedBefore != nil {
			q.Set("createdBefore", p.CreatedBefore.Format(time.RFC3339Nano))
		}
		if p.CreatedAfter != nil {
			q.Set("createdAfter", p.CreatedAfter.Format(time.RFC3339Nano))
		}
	}
	return websetsDo[Page[Event]](ctx, s.c, http.MethodGet, "/events", q, nil)
}

// Get returns one event.
func (s *WebsetEventsService) Get(ctx context.Context, id string) (*Event, error) {
	return websetsDo[Event](ctx, s.c, http.MethodGet, "/events/"+pathEscape(id), nil, nil)
}
