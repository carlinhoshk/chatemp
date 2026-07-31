package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"chatemp/internal/cleanup"
	"chatemp/internal/identity"
	"chatemp/internal/models"
	"chatemp/internal/store"
	"chatemp/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

func (h *Handlers) WebSocket(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	user := r.URL.Query().Get("user")
	if code == "" || user == "" {
		writeError(w, http.StatusBadRequest, "code and user are required")
		return
	}

	ctx := context.Background()
	room, err := h.store.RoomByCode(ctx, code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "sala não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if room.Expired(time.Now()) {
		writeError(w, http.StatusGone, "sala expirada")
		return
	}

	senderHash := identity.Hash(user)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(conn, room.ID, senderHash)
	h.hub.Join(room.ID, client)
	defer func() {
		h.hub.Leave(room.ID, client)
		client.Close()
	}()

	go client.WritePump()

	msgs, err := h.store.ListMessages(ctx, room.ID, 100)
	if err != nil {
		log.Printf("list messages: %v", err)
	}
	client.SendPayload(ws.Payload{
		Type:     "welcome",
		Room:     &room,
		Messages: msgs,
		Users:    h.hub.Members(room.ID),
		Self:     &ws.Self{Hash: senderHash, Short: identity.Short(senderHash)},
	})
	h.hub.Broadcast(room.ID, ws.Payload{Type: "users", Users: h.hub.Members(room.ID)})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if room.Expired(time.Now()) {
			h.hub.Broadcast(room.ID, ws.Payload{Type: "room_expired"})
			return
		}
		var in ws.Incoming
		if err := json.Unmarshal(data, &in); err != nil {
			continue
		}
		switch in.Type {
		case "chat":
			h.handleChat(ctx, room, senderHash, in.Content)
		case "media":
			h.handleMedia(ctx, room, senderHash, in.MessageID)
		case "viewed":
			h.handleViewed(ctx, in.MessageID)
		}
	}
}

func (h *Handlers) handleChat(ctx context.Context, room models.Room, senderHash, content string) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 4000 {
		return
	}
	m := models.Message{
		ID:         identity.NewID(),
		RoomID:     room.ID,
		SenderHash: senderHash,
		Kind:       "text",
		Body:       content,
		CreatedAt:  time.Now().UTC(),
	}
	if err := h.store.InsertMessage(ctx, m); err != nil {
		log.Printf("insert message: %v", err)
		return
	}
	m.SenderShort = identity.Short(senderHash)
	h.hub.Broadcast(room.ID, ws.Payload{Type: "message", Message: &m})
}

func (h *Handlers) handleMedia(ctx context.Context, room models.Room, senderHash, messageID string) {
	if messageID == "" {
		return
	}
	m, err := h.store.MessageByID(ctx, messageID)
	if err != nil || m.RoomID != room.ID || m.SenderHash != senderHash || (m.Kind != "image" && m.Kind != "video") {
		return
	}
	m.SenderShort = identity.Short(m.SenderHash)
	h.hub.Broadcast(room.ID, ws.Payload{Type: "message", Message: &m})
}

func (h *Handlers) handleViewed(ctx context.Context, messageID string) {
	if messageID == "" {
		return
	}
	m, ok, err := h.store.MarkViewed(ctx, messageID, time.Now().UTC())
	if err != nil || !ok {
		return
	}
	time.AfterFunc(h.cfg.EphemeralTTL, func() {
		cleanup.DeleteMedia(context.Background(), h.store, h.hub, h.storage, m)
	})
}
