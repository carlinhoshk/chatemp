package handlers_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"chatemp/internal/config"
	"chatemp/internal/handlers"
	"chatemp/internal/storage"
	"chatemp/internal/store"
	"chatemp/internal/ws"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{
		Port:           "0",
		DBPath:         filepath.Join(dir, "test.db"),
		UploadDir:      filepath.Join(dir, "uploads"),
		RoomTTL:        time.Hour,
		EphemeralTTL:   2 * time.Second,
		MediaSweep:     time.Minute,
		RoomPurge:      time.Minute,
		MaxUploadBytes: 10 << 20,
	}
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	fs, err := storage.New(cfg.UploadDir)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	hub := ws.NewHub()
	h := handlers.New(cfg, st, hub, fs)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/rooms", h.CreateRoom)
	mux.HandleFunc("GET /api/rooms/{code}", h.GetRoom)
	mux.HandleFunc("GET /api/rooms/{code}/messages", h.ListMessages)
	mux.HandleFunc("POST /api/rooms/{code}/media", h.UploadMedia)
	mux.HandleFunc("GET /api/media/{id}", h.ServeMedia)
	mux.HandleFunc("GET /ws", h.WebSocket)

	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		st.Close()
	})
	return srv, st
}

func createRoom(t *testing.T, srv *httptest.Server, user string) string {
	t.Helper()
	req, _ := http.NewRequest("POST", srv.URL+"/api/rooms", nil)
	req.Header.Set("X-User", user)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create room status: %d", res.StatusCode)
	}
	var room struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(res.Body).Decode(&room); err != nil {
		t.Fatalf("decode room: %v", err)
	}
	return room.Code
}

func dialWS(t *testing.T, srv *httptest.Server, code, user string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?code=" + code + "&user=" + user
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readPayload(t *testing.T, conn *websocket.Conn) ws.Payload {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws: %v", err)
	}
	var p ws.Payload
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("decode ws payload: %v", err)
	}
	return p
}

func TestChatFlow(t *testing.T) {
	srv, _ := newTestServer(t)
	code := createRoom(t, srv, "alice")

	alice := dialWS(t, srv, code, "alice")
	// alice welcome
	w := readPayload(t, alice)
	if w.Type != "welcome" || w.Room == nil || w.Room.Code != code {
		t.Fatalf("bad welcome: %+v", w)
	}

	bob := dialWS(t, srv, code, "bob")
	// bob welcome
	w = readPayload(t, bob)
	if w.Type != "welcome" {
		t.Fatalf("bad welcome: %+v", w)
	}
	// alice should eventually see both users
	for i := 0; i < 3; i++ {
		w = readPayload(t, alice)
		if w.Type == "users" && len(w.Users) == 2 {
			break
		}
		if i == 2 {
			t.Fatalf("expected users update with 2 members: %+v", w)
		}
	}
	// discard bob's own users broadcast before expecting the chat message
	if w = readPayload(t, bob); w.Type != "users" {
		t.Fatalf("expected bob users update: %+v", w)
	}

	// alice sends a text message
	if err := alice.WriteJSON(ws.Incoming{Type: "chat", Content: "olá temp!"}); err != nil {
		t.Fatalf("write chat: %v", err)
	}

	got := readPayload(t, bob)
	if got.Type != "message" || got.Message == nil {
		t.Fatalf("expected message payload, got %+v", got)
	}
	if got.Message.Kind != "text" || got.Message.Body != "olá temp!" {
		t.Fatalf("bad message: %+v", got.Message)
	}
	if got.Message.SenderShort == "" || len(got.Message.SenderShort) < 9 {
		t.Fatalf("missing sender short hash: %+v", got.Message)
	}
}

func TestExpiredRoomGone(t *testing.T) {
	srv, st := newTestServer(t)
	code := createRoom(t, srv, "alice")

	// force expiry
	if err := st.ExpireAllRooms(context.Background()); err != nil {
		t.Fatalf("expire rooms: %v", err)
	}

	res, err := http.Get(srv.URL + "/api/rooms/" + code)
	if err != nil {
		t.Fatalf("get room: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusGone {
		t.Fatalf("expected 410, got %d", res.StatusCode)
	}
}

// tiny 1x1 transparent PNG
var testPNG = func() []byte {
	b, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	if err != nil {
		panic(err)
	}
	return b
}()

func TestMediaUploadAndServe(t *testing.T) {
	srv, _ := newTestServer(t)
	code := createRoom(t, srv, "alice")

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "pixel.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(testPNG); err != nil {
		t.Fatalf("write png: %v", err)
	}
	mw.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/rooms/"+code+"/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-User", "alice")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(res.Body)
		t.Fatalf("upload status %d: %s", res.StatusCode, data)
	}
	var msg struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
		Mime string `json:"mime"`
	}
	if err := json.NewDecoder(res.Body).Decode(&msg); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	if msg.Kind != "image" || msg.Mime != "image/png" {
		t.Fatalf("bad media: %+v", msg)
	}

	// fetch it back
	media, err := http.Get(srv.URL + "/api/media/" + msg.ID)
	if err != nil {
		t.Fatalf("get media: %v", err)
	}
	defer media.Body.Close()
	if media.StatusCode != http.StatusOK {
		t.Fatalf("get media status: %d", media.StatusCode)
	}
	if ct := media.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content type: %s", ct)
	}
	got, _ := io.ReadAll(media.Body)
	if !bytes.Equal(got, testPNG) {
		t.Fatalf("media bytes mismatch (%d vs %d)", len(got), len(testPNG))
	}
}

func TestMediaBroadcast(t *testing.T) {
	srv, _ := newTestServer(t)
	code := createRoom(t, srv, "alice")
	alice := dialWS(t, srv, code, "alice")
	_ = readPayload(t, alice) // welcome
	bob := dialWS(t, srv, code, "bob")
	_ = readPayload(t, bob) // welcome
	for i := 0; i < 2; i++ {
		_ = readPayload(t, alice) // users updates
	}
	_ = readPayload(t, bob) // users update

	// upload as alice, then announce over ws
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "pixel.png")
	fw.Write(testPNG)
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/rooms/"+code+"/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-User", "alice")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	var uploaded struct {
		ID string `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&uploaded)
	res.Body.Close()

	if err := alice.WriteJSON(ws.Incoming{Type: "media", MessageID: uploaded.ID}); err != nil {
		t.Fatalf("write media: %v", err)
	}
	got := readPayload(t, bob)
	if got.Type != "message" || got.Message == nil || got.Message.ID != uploaded.ID {
		t.Fatalf("expected media message, got %+v", got)
	}
	if got.Message.Kind != "image" || got.Message.Mime != "image/png" {
		t.Fatalf("bad media message: %+v", got.Message)
	}
}
