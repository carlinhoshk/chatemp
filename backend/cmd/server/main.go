package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chatemp/internal/cleanup"
	"chatemp/internal/config"
	"chatemp/internal/handlers"
	"chatemp/internal/storage"
	"chatemp/internal/store"
	"chatemp/internal/ws"
)

func main() {
	cfg := config.Load()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	fs, err := storage.New(cfg.UploadDir)
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	hub := ws.NewHub()
	h := handlers.New(cfg, st, hub, fs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cleanup.Run(ctx, cfg, st, hub, fs)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/rooms", h.CreateRoom)
	mux.HandleFunc("GET /api/rooms/{code}", h.GetRoom)
	mux.HandleFunc("GET /api/rooms/{code}/messages", h.ListMessages)
	mux.HandleFunc("GET /ws", h.WebSocket)

	if cfg.StaticDir != "" {
		if _, err := os.Stat(filepath.Join(cfg.StaticDir, "index.html")); err == nil {
			mux.Handle("/", spaHandler(cfg.StaticDir))
			log.Printf("serving static frontend from %s", cfg.StaticDir)
		}
	}

	addr := ":" + cfg.Port
	log.Printf("chatemp listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
