package daemon

import (
	"sync"

	"github.com/Twarga/yeast/internal/app"
)

const eventBufferSize = 512

// Hub is a per-project event broadcaster: recent events are buffered for
// late-joining SSE clients and live events fan out to all subscribers.
type Hub struct {
	mu   sync.Mutex
	buf  []app.Event
	subs map[chan app.Event]struct{}
}

func newHub() *Hub {
	return &Hub{subs: map[chan app.Event]struct{}{}}
}

// Publish appends to the ring buffer and notifies all subscribers.
func (h *Hub) Publish(e app.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf = append(h.buf, e)
	if len(h.buf) > eventBufferSize {
		h.buf = append([]app.Event(nil), h.buf[len(h.buf)-eventBufferSize:]...)
	}
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// Subscribe returns buffered events plus a live channel. Call the returned
// cancel func to unsubscribe.
func (h *Hub) Subscribe() ([]app.Event, <-chan app.Event, func()) {
	ch := make(chan app.Event, 64)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subs[ch] = struct{}{}
	buffered := append([]app.Event(nil), h.buf...)
	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subs, ch)
		close(ch)
	}
	return buffered, ch, cancel
}

// Sink returns an app.EventSink that publishes into this hub.
func (h *Hub) Sink() app.EventSink {
	return func(e app.Event) { h.Publish(e) }
}
