package handlers

import (
	"encoding/json"
	"net/http"

	"chatemp/internal/config"
	"chatemp/internal/storage"
	"chatemp/internal/store"
	"chatemp/internal/ws"
)

type Handlers struct {
	cfg     config.Config
	store   *store.Store
	hub     *ws.Hub
	storage *storage.Storage
}

func New(cfg config.Config, st *store.Store, hub *ws.Hub, fs *storage.Storage) *Handlers {
	return &Handlers{cfg: cfg, store: st, hub: hub, storage: fs}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
