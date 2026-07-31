package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"chatemp/internal/identity"
	"chatemp/internal/models"
	"chatemp/internal/store"
)

func (h *Handlers) UploadMedia(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("X-User")
	if user == "" {
		writeError(w, http.StatusBadRequest, "missing X-User header")
		return
	}
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

	ephemeral := false
	switch r.URL.Query().Get("ephemeral") {
	case "1", "true", "on":
		ephemeral = true
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxUploadBytes)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "arquivo ausente")
		return
	}
	defer file.Close()

	if header.Size > h.cfg.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "arquivo muito grande")
		return
	}

	sniff := make([]byte, 512)
	n, _ := io.ReadFull(file, sniff)
	ctype := http.DetectContentType(sniff[:n])
	kind, ext := mediaKind(ctype)
	if kind == "" {
		writeError(w, http.StatusUnsupportedMediaType, "tipo de mídia não suportado")
		return
	}

	id := identity.NewID()
	m := models.Message{
		ID:         id,
		RoomID:     room.ID,
		SenderHash: identity.Hash(user),
		Kind:       kind,
		Mime:       ctype,
		SizeBytes:  header.Size,
		CreatedAt:  time.Now().UTC(),
	}
	if ephemeral {
		m.IsEphemeral = true
		m.TTLSeconds = int(h.cfg.EphemeralTTL.Seconds())
	}

	rel, err := h.storage.Save(room.ID, id, ext, io.MultiReader(bytes.NewReader(sniff[:n]), file))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao salvar arquivo")
		return
	}
	m.MediaPath = rel
	if err := h.store.InsertMessage(r.Context(), m); err != nil {
		h.storage.Delete(rel)
		writeError(w, http.StatusInternalServerError, "falha ao salvar mensagem")
		return
	}
	m.SenderShort = identity.Short(m.SenderHash)
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handlers) ServeMedia(w http.ResponseWriter, r *http.Request) {
	m, err := h.store.MessageByID(r.Context(), r.PathValue("id"))
	if err != nil || m.MediaPath == "" {
		writeError(w, http.StatusNotFound, "mídia não encontrada")
		return
	}
	room, err := h.store.RoomByID(r.Context(), m.RoomID)
	if err != nil || room.Expired(time.Now()) {
		writeError(w, http.StatusNotFound, "mídia não encontrada")
		return
	}
	path := h.storage.Path(m.MediaPath)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "mídia não encontrada")
		return
	}
	w.Header().Set("Content-Type", m.Mime)
	http.ServeFile(w, r, path)
}

func mediaKind(ctype string) (kind, ext string) {
	switch {
	case strings.HasPrefix(ctype, "image/"):
		return "image", extFor(ctype)
	case strings.HasPrefix(ctype, "video/"):
		return "video", extFor(ctype)
	}
	return "", ""
}

func extFor(ctype string) string {
	switch ctype {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/ogg":
		return ".ogv"
	case "video/quicktime":
		return ".mov"
	case "video/x-msvideo":
		return ".avi"
	case "video/mpeg":
		return ".mpeg"
	}
	return ""
}
