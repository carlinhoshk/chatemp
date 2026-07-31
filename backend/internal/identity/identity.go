package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// NewID returns a random 32-char hex identifier.
func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// NewCode returns an 8-char room code avoiding ambiguous characters.
func NewCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}

// Hash derives the server-side identity hash from a client-provided id.
func Hash(user string) string {
	sum := sha256.Sum256([]byte(user))
	return hex.EncodeToString(sum[:])
}

// Short returns a display-friendly abbreviation like f3a8…9c41.
func Short(hash string) string {
	if len(hash) < 8 {
		return hash
	}
	return hash[:4] + "…" + hash[len(hash)-4:]
}
