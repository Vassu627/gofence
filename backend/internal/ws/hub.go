package ws

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu       sync.RWMutex
	clients  map[*client]struct{}
	upgrader websocket.Upgrader
	apiKey   string
}

type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewHub(apiKey string) *Hub {
	return &Hub{
		clients: map[*client]struct{}{},
		apiKey:  apiKey,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	if h.apiKey != "" {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}
		if subtle.ConstantTimeCompare([]byte(key), []byte(h.apiKey)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}
	c := &client{conn: conn}
	h.add(c)
	log.Printf("websocket connected clients=%d", h.count())

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.remove(c)
		log.Printf("websocket disconnected clients=%d", h.count())
	}()
	go c.keepalive(ctx)

	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetReadLimit(1024)
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// Broadcast sends one alert to every connected client.
// Call it from a goroutine so the HTTP handler can return immediately.
func (h *Hub) Broadcast(v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		log.Printf("websocket marshal: %v", err)
		return
	}
	h.mu.RLock()
	snapshot := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		snapshot = append(snapshot, c)
	}
	h.mu.RUnlock()
	for _, c := range snapshot {
		if err := c.write(websocket.TextMessage, payload); err != nil {
			h.remove(c)
		}
	}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	_ = c.conn.Close()
}

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *client) write(messageType int, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteMessage(messageType, payload)
}

func (c *client) keepalive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
