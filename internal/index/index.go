package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/toshism/tnotes/internal/config"
	"github.com/toshism/tnotes/internal/note"
)

// Index holds all note entries for fast lookup
type Index struct {
	Entries []note.IndexEntry `json:"entries"`
}

// File is the on-disk state of one note file.
type File struct {
	Path    string
	ModTime int64 // Unix nanoseconds
	Size    int64
}

// Changes lists how the note files on disk differ from the index.
type Changes struct {
	Added   []File
	Changed []File
	Removed []note.IndexEntry
}

// Empty reports whether the index matches the files on disk.
func (c Changes) Empty() bool {
	return len(c.Added) == 0 && len(c.Changed) == 0 && len(c.Removed) == 0
}

// Diff compares index entries against note files on disk, matching by path.
func Diff(entries []note.IndexEntry, files []File) Changes {
	indexed := make(map[string]note.IndexEntry, len(entries))
	for _, e := range entries {
		indexed[e.Path] = e
	}

	var c Changes
	onDisk := make(map[string]bool, len(files))
	for _, f := range files {
		onDisk[f.Path] = true
		e, ok := indexed[f.Path]
		if !ok {
			c.Added = append(c.Added, f)
		} else if e.ModTime != f.ModTime || e.Size != f.Size {
			c.Changed = append(c.Changed, f)
		}
	}
	for _, e := range entries {
		if !onDisk[e.Path] {
			c.Removed = append(c.Removed, e)
		}
	}
	return c
}

// Update is what a search index must apply to match a refreshed index.
type Update struct {
	Upserted []note.IndexEntry
	Removed  []string // IDs no longer in the index
}

// Empty reports whether the refresh changed nothing.
func (u Update) Empty() bool {
	return len(u.Upserted) == 0 && len(u.Removed) == 0
}

// Load reads the index from disk
func Load() (*Index, error) {
	return LoadFor(config.NotesDir)
}

// LoadFor reads the index for a notes directory, returning an empty index if
// none has been built yet.
func LoadFor(notesDir string) (*Index, error) {
	data, err := os.ReadFile(config.IndexFileFor(notesDir))
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
	return idx.SaveFor(config.NotesDir)
}

// SaveFor writes the index for a notes directory. It writes a temporary file
// and renames it, so a concurrent reader never sees a partial index.
func (idx *Index) SaveFor(notesDir string) error {
	indexDir := config.IndexDirFor(notesDir)
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(indexDir, "index-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), config.IndexFileFor(notesDir))
}

// Rebuild scans the notes directory and rebuilds the index from scratch
func Rebuild() (*Index, error) {
	return RebuildFor(config.NotesDir)
}

// RebuildFor builds the index for a notes directory from scratch.
func RebuildFor(notesDir string) (*Index, error) {
	idx := &Index{Entries: []note.IndexEntry{}}
	if _, err := Refresh(idx, notesDir); err != nil {
		return nil, err
	}
	return idx, nil
}

// Refresh brings idx up to date with the note files under notesDir and returns
// what a search index must apply to match. Notes that fail to parse are left
// out of the index.
func Refresh(idx *Index, notesDir string) (Update, error) {
	files, err := scan(notesDir)
	if err != nil {
		return Update{}, err
	}
	changes := Diff(idx.Entries, files)
	if changes.Empty() {
		return Update{}, nil
	}

	stale := make(map[string]bool)
	for _, f := range changes.Changed {
		stale[f.Path] = true
	}
	for _, e := range changes.Removed {
		stale[e.Path] = true
	}

	affected := make(map[string]bool)
	entries := make([]note.IndexEntry, 0, len(idx.Entries)+len(changes.Added))
	for _, e := range idx.Entries {
		if stale[e.Path] {
			affected[e.ID] = true
			continue
		}
		entries = append(entries, e)
	}
	for _, files := range [][]File{changes.Added, changes.Changed} {
		for _, f := range files {
			entry, err := parseEntry(f)
			if err != nil {
				continue
			}
			entries = append(entries, entry)
			affected[entry.ID] = true
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	idx.Entries = entries

	return updateFor(idx, affected), nil
}

// scan lists the note files under notesDir. A missing or unreadable notes
// directory is an error, so it never reads as every note being deleted.
func scan(notesDir string) ([]File, error) {
	root := config.ResolvedNotesDirFor(notesDir)
	var files []File
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if path == root {
				return err
			}
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

		files = append(files, File{Path: path, ModTime: info.ModTime().UnixNano(), Size: info.Size()})
		return nil
	})
	return files, err
}

func parseEntry(f File) (note.IndexEntry, error) {
	n, _, err := note.ParseFile(f.Path)
	if err != nil {
		return note.IndexEntry{}, err
	}
	entry := n.ToIndexEntry()
	// Keep the scan's stat, so an edit that lands after it reads as a change
	// on the next refresh.
	entry.ModTime = f.ModTime
	entry.Size = f.Size
	return entry, nil
}

// updateFor upserts each affected ID that is still indexed and removes the
// rest, in ID order.
func updateFor(idx *Index, affected map[string]bool) Update {
	byID := make(map[string]note.IndexEntry, len(idx.Entries))
	for _, e := range idx.Entries {
		if _, seen := byID[e.ID]; !seen {
			byID[e.ID] = e
		}
	}

	ids := make([]string, 0, len(affected))
	for id := range affected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var u Update
	for _, id := range ids {
		if e, ok := byID[id]; ok {
			u.Upserted = append(u.Upserted, e)
		} else {
			u.Removed = append(u.Removed, id)
		}
	}
	return u
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
