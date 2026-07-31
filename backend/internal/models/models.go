package models

import "time"

type Room struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Creator   string    `json:"creator"`
}

func (r Room) Expired(now time.Time) bool {
	return now.After(r.ExpiresAt)
}

type Message struct {
	ID          string    `json:"id"`
	RoomID      string    `json:"-"`
	SenderHash  string    `json:"sender"`
	SenderShort string    `json:"sender_short"`
	Kind        string    `json:"kind"`
	Body        string    `json:"body,omitempty"`
	MediaPath   string    `json:"-"`
	Mime        string    `json:"mime,omitempty"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	IsEphemeral bool      `json:"is_ephemeral"`
	Viewed      bool      `json:"viewed"`
	TTLSeconds  int       `json:"ttl_seconds,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
