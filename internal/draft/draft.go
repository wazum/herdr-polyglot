// Package draft keeps an unfinished prompt for the pane it was written for, so
// closing the popup costs nothing and the thought can be picked up again.
package draft

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store is a directory of drafts, one per pane.
type Store struct{ directory string }

// NewStore takes the directory herdr set aside for this plugin's state. An empty
// directory means drafts are simply not kept.
func NewStore(directory string) Store {
	return Store{directory: directory}
}

func (s Store) For(target string) Slot {
	if s.directory == "" {
		return Slot{}
	}
	return Slot{path: filepath.Join(s.directory, fileName(target))}
}

// Slot is the draft belonging to one pane.
type Slot struct{ path string }

func (s Slot) Load() string {
	if s.path == "" {
		return ""
	}
	kept, err := os.ReadFile(s.path)
	if err != nil {
		return ""
	}
	return string(kept)
}

// Save keeps the draft, or removes it when there is nothing left to keep.
func (s Slot) Save(text string) error {
	if s.path == "" {
		return nil
	}
	if strings.TrimSpace(text) == "" {
		return s.Clear()
	}
	// A draft is unfinished thinking about the author's own work.
	if err := os.WriteFile(s.path, []byte(text), 0o600); err != nil {
		return fmt.Errorf("keeping the draft: %w", err)
	}
	return nil
}

func (s Slot) Clear() error {
	if s.path == "" {
		return nil
	}
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("forgetting the draft: %w", err)
	}
	return nil
}

// fileName turns a pane id into one path component. Hashing keeps a target that
// contains separators from pointing anywhere but the store.
func fileName(target string) string {
	digest := sha256.Sum256([]byte(target))
	return "draft-" + hex.EncodeToString(digest[:8])
}
