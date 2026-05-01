package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/toshism/tnotes/internal/config"
	"github.com/toshism/tnotes/internal/note"
)

// Index holds all note entries for fast lookup
type Index struct {
	Entries []note.IndexEntry `json:"entries"`
}

// Load reads the index from disk
func Load() (*Index, error) {
	indexFile := config.IndexFile()

	data, err := os.ReadFile(indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Entries: []note.IndexEntry{}}, nil
		}
		return nil, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	return &idx, nil
}

// Save writes the index to disk
func (idx *Index) Save() error {
	indexDir := config.IndexDir()
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(config.IndexFile(), data, 0644)
}

// Rebuild scans the notes directory and rebuilds the index from scratch
func Rebuild() (*Index, error) {
	notesDir := config.ResolvedNotesDirFor(config.NotesDir)
	idx := &Index{Entries: []note.IndexEntry{}}

	err := filepath.Walk(notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip .tnotes directory
		if info.IsDir() && info.Name() == ".tnotes" {
			return filepath.SkipDir
		}

		// Only process .md files
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		n, _, err := note.ParseFile(path)
		if err != nil {
			// Skip files that can't be parsed
			return nil
		}

		idx.Entries = append(idx.Entries, n.ToIndexEntry())
		return nil
	})

	if err != nil {
		return nil, err
	}

	return idx, nil
}

// AddEntry adds a new entry to the index
func (idx *Index) AddEntry(entry note.IndexEntry) {
	// Remove existing entry with same ID if it exists
	idx.RemoveByID(entry.ID)
	idx.Entries = append(idx.Entries, entry)
}

// RemoveByID removes an entry by ID
func (idx *Index) RemoveByID(id string) {
	filtered := make([]note.IndexEntry, 0, len(idx.Entries))
	for _, e := range idx.Entries {
		if e.ID != id {
			filtered = append(filtered, e)
		}
	}
	idx.Entries = filtered
}

// FindByID returns an entry by ID
func (idx *Index) FindByID(id string) *note.IndexEntry {
	for i := range idx.Entries {
		if idx.Entries[i].ID == id {
			return &idx.Entries[i]
		}
	}
	return nil
}

// AllTags returns all unique tags with their counts
func (idx *Index) AllTags() map[string]int {
	tags := make(map[string]int)
	for _, e := range idx.Entries {
		for _, t := range e.Tags {
			tags[t]++
		}
	}
	return tags
}
