package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"chatemp/internal/identity"
	"chatemp/internal/models"
)

const msLayout = "2006-01-02T15:04:05.000Z"

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: single writer
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS rooms (
    id          TEXT PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    created_at  TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    creator_hash TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS messages (
    id                    TEXT PRIMARY KEY,
    room_id               TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    sender_hash           TEXT NOT NULL,
    kind                  TEXT NOT NULL,
    body                  TEXT,
    media_path            TEXT,
    mime                  TEXT,
    size_bytes            INTEGER,
    is_ephemeral          INTEGER NOT NULL DEFAULT 0,
    viewed_at             TEXT,
    delete_after_seconds  INTEGER,
    created_at            TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_room ON messages(room_id, created_at);
CREATE INDEX IF NOT EXISTS idx_rooms_expires ON rooms(expires_at);
CREATE INDEX IF NOT EXISTS idx_messages_ephemeral ON messages(is_ephemeral, viewed_at);
`)
	return err
}

func formatTime(t time.Time) string {
	return t.UTC().Format(msLayout)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(msLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// ---- rooms ----

func (s *Store) CreateRoom(ctx context.Context, r models.Room) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rooms (id, code, created_at, expires_at, creator_hash) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Code, formatTime(r.CreatedAt), formatTime(r.ExpiresAt), r.Creator)
	return err
}

func scanRoom(row *sql.Row) (models.Room, error) {
	var r models.Room
	var created, expires string
	err := row.Scan(&r.ID, &r.Code, &created, &expires, &r.Creator)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.CreatedAt = parseTime(created)
	r.ExpiresAt = parseTime(expires)
	return r, nil
}

func (s *Store) RoomByCode(ctx context.Context, code string) (models.Room, error) {
	return scanRoom(s.db.QueryRowContext(ctx,
		`SELECT id, code, created_at, expires_at, creator_hash FROM rooms WHERE code = ?`, code))
}

func (s *Store) RoomByID(ctx context.Context, id string) (models.Room, error) {
	return scanRoom(s.db.QueryRowContext(ctx,
		`SELECT id, code, created_at, expires_at, creator_hash FROM rooms WHERE id = ?`, id))
}

// ---- messages ----

const messageCols = `id, room_id, sender_hash, kind, body, media_path, mime, size_bytes, is_ephemeral, viewed_at, delete_after_seconds, created_at`

func scanMessage(row interface{ Scan(...any) error }) (models.Message, error) {
	var m models.Message
	var body, mediaPath, mime, viewedAt sql.NullString
	var size sql.NullInt64
	var ttl sql.NullInt64
	var created string
	err := row.Scan(&m.ID, &m.RoomID, &m.SenderHash, &m.Kind, &body, &mediaPath, &mime, &size, &m.IsEphemeral, &viewedAt, &ttl, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, err
	}
	m.Body = body.String
	m.MediaPath = mediaPath.String
	m.Mime = mime.String
	m.SizeBytes = size.Int64
	m.Viewed = viewedAt.Valid
	m.TTLSeconds = int(ttl.Int64)
	m.CreatedAt = parseTime(created)
	m.SenderShort = identity.Short(m.SenderHash)
	return m, nil
}

func (s *Store) InsertMessage(ctx context.Context, m models.Message) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, room_id, sender_hash, kind, body, media_path, mime, size_bytes, is_ephemeral, viewed_at, delete_after_seconds, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.RoomID, m.SenderHash, m.Kind, nullStr(m.Body), nullStr(m.MediaPath), nullStr(m.Mime),
		m.SizeBytes, boolInt(m.IsEphemeral), nullStr(""), intOrNil(m.TTLSeconds), formatTime(m.CreatedAt))
	return err
}

func (s *Store) MessageByID(ctx context.Context, id string) (models.Message, error) {
	return scanMessage(s.db.QueryRowContext(ctx, `SELECT `+messageCols+` FROM messages WHERE id = ?`, id))
}

func (s *Store) ListMessages(ctx context.Context, roomID string, limit int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+messageCols+` FROM messages WHERE room_id = ? ORDER BY created_at DESC LIMIT ?`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Message, 0, limit)
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// reverse to chronological order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// MarkViewed sets viewed_at once (idempotent) and returns the message if it was newly marked.
func (s *Store) MarkViewed(ctx context.Context, id string, at time.Time) (models.Message, bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE messages SET viewed_at = ? WHERE id = ? AND is_ephemeral = 1 AND viewed_at IS NULL`,
		formatTime(at), id)
	if err != nil {
		return models.Message{}, false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return models.Message{}, false, err
	}
	if n == 0 {
		return models.Message{}, false, nil
	}
	m, err := s.MessageByID(ctx, id)
	if err != nil {
		return models.Message{}, false, err
	}
	return m, true, nil
}

func (s *Store) DeleteMessage(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, id)
	return err
}

// ConsumedMedia returns ephemeral messages whose post-view timer has elapsed.
func (s *Store) ConsumedMedia(ctx context.Context, before time.Time) ([]models.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+messageCols+` FROM messages WHERE is_ephemeral = 1 AND viewed_at IS NOT NULL AND viewed_at <= ?`,
		formatTime(before))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) MediaPathsByRoom(ctx context.Context, roomID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT media_path FROM messages WHERE room_id = ? AND media_path IS NOT NULL AND media_path != ''`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ---- expiry ----

func (s *Store) ExpiredRooms(ctx context.Context, now time.Time) ([]models.Room, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code, created_at, expires_at, creator_hash FROM rooms WHERE expires_at <= ?`, formatTime(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Room
	for rows.Next() {
		var r models.Room
		var created, expires string
		if err := rows.Scan(&r.ID, &r.Code, &created, &expires, &r.Creator); err != nil {
			return nil, err
		}
		r.CreatedAt = parseTime(created)
		r.ExpiresAt = parseTime(expires)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteRoom(ctx context.Context, roomID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rooms WHERE id = ?`, roomID)
	return err
}

// ExpireAllRooms forces every room to be already expired (used by tests/ops).
func (s *Store) ExpireAllRooms(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE rooms SET expires_at = ?`, formatTime(time.Now().Add(-time.Second)))
	return err
}

func (s *Store) Count() (rooms int, messages int) {
	s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM rooms), (SELECT COUNT(*) FROM messages)`).Scan(&rooms, &messages)
	return
}

// ---- helpers ----

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func intOrNil(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
