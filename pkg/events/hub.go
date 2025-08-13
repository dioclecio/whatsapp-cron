package events

import (
	"encoding/json"
	"github.com/olahol/gows"
	"net/http"
	"sync"
)

type EventType string

const (
	EventQRReady     EventType = "QR_READY"
	EventLoggedIn    EventType = "LOGGED_IN"
	EventMessageSent EventType = "MESSAGE_SENT"
	EventError       EventType = "ERROR"
)

type Event struct {
	Type    EventType   `json:"type"`
	Session string      `json:"session,omitempty"`
	To      string      `json:"to,omitempty"`
	ID      int         `json:"id,omitempty"`
	Scope   string      `json:"scope,omitempty"`
	Message string      `json:"message,omitempty"`
}

type Hub struct {
	hub  *gows.Hub
	lock sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		hub: gows.NewHub(),
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()

	ws, err := gows.Upgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.hub.Add(ws)
}

func (h *Hub) Publish(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	h.hub.Broadcast(data)
	return nil
}

func (h *Hub) PublishQRReady(session string) error {
	return h.Publish(Event{
		Type:    EventQRReady,
		Session: session,
	})
}

func (h *Hub) PublishLoggedIn(session string) error {
	return h.Publish(Event{
		Type:    EventLoggedIn,
		Session: session,
	})
}

func (h *Hub) PublishMessageSent(to string, id int) error {
	return h.Publish(Event{
		Type: EventMessageSent,
		To:   to,
		ID:   id,
	})
}

func (h *Hub) PublishError(scope, message string) error {
	return h.Publish(Event{
		Type:    EventError,
		Scope:   scope,
		Message: message,
	})
}
