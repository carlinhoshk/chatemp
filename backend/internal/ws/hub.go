package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"chatemp/internal/models"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendBuffer = 64
)

// Payload is a server -> client message.
type Payload struct {
	Type      string           `json:"type"`
	Message   *models.Message  `json:"message,omitempty"`
	MessageID string           `json:"message_id,omitempty"`
	Room      *models.Room     `json:"room,omitempty"`
	Messages  []models.Message `json:"messages,omitempty"`
	Users     []string         `json:"users,omitempty"`
	Self      *Self            `json:"self,omitempty"`
	Error     string           `json:"error,omitempty"`
}

type Self struct {
	Hash  string `json:"hash"`
	Short string `json:"short"`
}

// Incoming is a client -> server message.
type Incoming struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	MessageID string `json:"message_id"`
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once

	Room string
	Hash string
}

func NewClient(conn *websocket.Conn, room, hash string) *Client {
	return &Client{
		conn: conn,
		send: make(chan []byte, sendBuffer),
		done: make(chan struct{}),
		Room: room,
		Hash: hash,
	}
}

func (c *Client) SendPayload(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) Close() {
	c.once.Do(func() { close(c.done) })
}

func (c *Client) WritePump() {
	defer c.conn.Close()
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

// Hub tracks live connections per room (keyed by internal room id).
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: map[string]map[*Client]struct{}{}}
}

func (h *Hub) Join(roomID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = map[*Client]struct{}{}
	}
	h.rooms[roomID][c] = struct{}{}
}

func (h *Hub) Leave(roomID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set := h.rooms[roomID]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) Broadcast(roomID string, payload any) {
	h.mu.RLock()
	set := h.rooms[roomID]
	clients := make([]*Client, 0, len(set))
	for c := range set {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		c.SendPayload(payload)
	}
}

// Members returns the distinct sender hashes currently connected to a room.
func (h *Hub) Members(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := map[string]struct{}{}
	var out []string
	for c := range h.rooms[roomID] {
		if _, ok := seen[c.Hash]; ok {
			continue
		}
		seen[c.Hash] = struct{}{}
		out = append(out, c.Hash)
	}
	return out
}

func (h *Hub) Rooms() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.rooms))
	for id := range h.rooms {
		out = append(out, id)
	}
	return out
}

// CloseRoom disconnects every client in a room.
func (h *Hub) CloseRoom(roomID string) {
	h.mu.Lock()
	set := h.rooms[roomID]
	delete(h.rooms, roomID)
	h.mu.Unlock()
	for c := range set {
		c.Close()
	}
}
