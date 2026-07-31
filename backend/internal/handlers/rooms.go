package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"chatemp/internal/identity"
	"chatemp/internal/models"
	"chatemp/internal/store"
)

func (h *Handlers) CreateRoom(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User")
	if user == "" {
		writeError(w, http.StatusBadRequest, "missing X-User header")
		return
	}
	now := time.Now().UTC()
	room := models.Room{
		ID:        identity.NewID(),
		Code:      identity.NewCode(),
		CreatedAt: now,
		ExpiresAt: now.Add(h.cfg.RoomTTL),
		Creator:   identity.Hash(user),
	}
	if err := h.store.CreateRoom(r.Context(), room); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create room")
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

func (h *Handlers) GetRoom(w http.ResponseWriter, r *http.Request) {
	room, err := h.store.RoomByCode(r.Context(), r.PathValue("code"))
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
	writeJSON(w, http.StatusOK, room)
}

func (h *Handlers) ListMessages(w http.ResponseWriter, r *http.Request) {
	room, err := h.store.RoomByCode(r.Context(), r.PathValue("code"))
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
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	msgs, err := h.store.ListMessages(r.Context(), room.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}
