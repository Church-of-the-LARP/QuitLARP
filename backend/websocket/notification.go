package websocket

import "encoding/json"

// Notifier pushes JSON notifications to users through a Hub.
type Notifier struct {
	hub *Hub
}

func NewNotifier(hub *Hub) *Notifier {
	return &Notifier{hub: hub}
}

type envelope struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// Send delivers {"type": kind, "data": data} to every open connection of the
// user and returns how many received it (0 if the user is offline).
func (n *Notifier) Send(userID int64, kind string, data any) (int, error) {
	payload, err := json.Marshal(envelope{Type: kind, Data: data})
	if err != nil {
		return 0, err
	}
	return n.hub.Send(userID, payload), nil
}
