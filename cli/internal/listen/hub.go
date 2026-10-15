package listen

import "sync"

type hub struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

func newHub() *hub {
	return &hub{subs: make(map[chan Event]struct{})}
}

func (h *hub) subscribe() chan Event {
	ch := make(chan Event, 64)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *hub) unsubscribe(ch chan Event) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *hub) broadcast(e Event) {
	// Non-blocking fan-out: snapshot subs under the lock, then send without
	// holding it. A slow subscriber with a full buffer (cap 64) is skipped
	// instead of stalling every other subscriber and the notifier.
	h.mu.Lock()
	subs := make([]chan Event, 0, len(h.subs))
	for ch := range h.subs {
		subs = append(subs, ch)
	}
	h.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- e:
		default:
		}
	}
}
