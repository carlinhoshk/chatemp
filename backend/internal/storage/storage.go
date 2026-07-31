package storage

import (
	"io"
	"os"
	"path/filepath"
)

type Storage struct {
	root string
}

func New(root string) (*Storage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Storage{root: root}, nil
}

// Save writes a media file under <root>/<roomID>/<messageID><ext> and returns its relative path.
func (s *Storage) Save(roomID, messageID, ext string, r io.Reader) (string, error) {
	dir := filepath.Join(s.root, roomID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := messageID + ext
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, r); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(roomID, name)), nil
}

func (s *Storage) Path(rel string) string {
	return filepath.Join(s.root, filepath.FromSlash(rel))
}

func (s *Storage) Delete(rel string) error {
	if rel == "" {
		return nil
	}
	return os.Remove(s.Path(rel))
}

func (s *Storage) DeleteRoom(roomID string) error {
	return os.RemoveAll(filepath.Join(s.root, roomID))
}
