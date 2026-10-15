package listen

import (
	"encoding/json"
	"time"
)

type Event struct {
	Topic string `json:"topic"`
	Repo  string `json:"repo"`
	At    string `json:"at"`
	ID    string `json:"id,omitempty"`
	Op    string `json:"op,omitempty"`
}

func NewEvent(topic, repo string) Event {
	return Event{
		Topic: topic,
		Repo:  repo,
		At:    time.Now().UTC().Format(time.RFC3339),
	}
}

func (e Event) JSONLine() []byte {
	b, _ := json.Marshal(e)
	return b
}

type Config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	PID   int    `json:"pid,omitempty"`
}
