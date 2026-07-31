package main

import (
	"log"
	"net/http"

	"chatemp/internal/config"
)

func main() {
	cfg := config.Load()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("chatemp"))
	})
	addr := ":" + cfg.Port
	log.Printf("chatemp listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
