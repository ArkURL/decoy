// Package storage persists reading progress to a JSON file.
package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Position is a width-independent reading position.
type Position struct {
	SpineIndex int `json:"spine"`
	Para       int `json:"para"`
	LineInPara int `json:"line"`
}

type bookState struct {
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	Position Position `json:"position"`
	Updated  int64    `json:"updatedAt"`
}

// Store holds progress for many books and writes them atomically.
type Store struct {
	path  string
	mu    sync.Mutex
	Books map[string]*bookState `json:"books"`
}

// DefaultPath returns the progress file location.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "cloak", "progress.json")
}

// NewEmpty returns an in-memory store that persists to path on Save.
func NewEmpty(path string) *Store {
	return &Store{path: path, Books: map[string]*bookState{}}
}

// Load reads the store from path. A missing file yields an empty store.
func Load(path string) (*Store, error) {
	s := &Store{path: path, Books: map[string]*bookState{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, s); err != nil {
			// Corrupt file: start fresh rather than failing to launch.
			s.Books = map[string]*bookState{}
		}
	}
	if s.Books == nil {
		s.Books = map[string]*bookState{}
	}
	return s, nil
}

func idFor(bookPath string) string {
	abs, err := filepath.Abs(bookPath)
	if err != nil {
		abs = bookPath
	}
	sum := sha1.Sum([]byte(abs))
	return hex.EncodeToString(sum[:])
}

// Get returns the stored position for a book, if any.
func (s *Store) Get(bookPath string) (Position, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, ok := s.Books[idFor(bookPath)]; ok {
		return b.Position, true
	}
	return Position{}, false
}

// Set records a position for a book.
func (s *Store) Set(bookPath, title string, pos Position) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Books[idFor(bookPath)] = &bookState{
		Path:     bookPath,
		Title:    title,
		Position: pos,
		Updated:  time.Now().Unix(),
	}
}

// Save writes the store atomically.
func (s *Store) Save() error {
	s.mu.Lock()
	data, err := json.MarshalIndent(s, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
