package index

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/toshism/tnotes/internal/note"
)

func TestDiff(t *testing.T) {
	indexed := []note.IndexEntry{
		{ID: "1", Path: "/n/same.md", ModTime: 100, Size: 10},
		{ID: "2", Path: "/n/touched.md", ModTime: 100, Size: 10},
		{ID: "3", Path: "/n/resized.md", ModTime: 100, Size: 10},
		{ID: "4", Path: "/n/gone.md", ModTime: 100, Size: 10},
	}
	onDisk := []File{
		{Path: "/n/same.md", ModTime: 100, Size: 10},
		{Path: "/n/touched.md", ModTime: 101, Size: 10},
		{Path: "/n/resized.md", ModTime: 100, Size: 11},
		{Path: "/n/new.md", ModTime: 100, Size: 10},
	}

	got := Diff(indexed, onDisk)

	want := Changes{
		Added:   []File{{Path: "/n/new.md", ModTime: 100, Size: 10}},
		Changed: []File{{Path: "/n/touched.md", ModTime: 101, Size: 10}, {Path: "/n/resized.md", ModTime: 100, Size: 11}},
		Removed: []note.IndexEntry{{ID: "4", Path: "/n/gone.md", ModTime: 100, Size: 10}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Diff() = %+v, want %+v", got, want)
	}
}

func TestDiffUnchangedIsEmpty(t *testing.T) {
	indexed := []note.IndexEntry{{ID: "1", Path: "/n/a.md", ModTime: 100, Size: 10}}
	onDisk := []File{{Path: "/n/a.md", ModTime: 100, Size: 10}}

	if got := Diff(indexed, onDisk); !got.Empty() {
		t.Fatalf("Diff() = %+v, want empty", got)
	}
}

func TestRefreshAfterAddEntryIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := writeNote(t, dir, "a.md", "a", "first body")
	n, _, err := note.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	idx := &Index{}
	idx.AddEntry(n.ToIndexEntry())

	update, err := Refresh(idx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !update.Empty() {
		t.Fatalf("Refresh() = %+v, want empty after AddEntry", update)
	}
}

func TestRefreshPicksUpNewEditedAndDeletedNotes(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "a", "first body")
	bPath := writeNote(t, dir, "b.md", "b", "second body")
	idx := &Index{}
	update, err := Refresh(idx, dir)
	if err != nil {
		t.Fatal(err)
	}
	assertUpdate(t, update, []string{"a", "b"}, nil)

	writeNote(t, dir, "a.md", "a", "first body, now longer")
	writeNote(t, dir, "c.md", "c", "third body")
	if err := os.Remove(bPath); err != nil {
		t.Fatal(err)
	}
	update, err = Refresh(idx, dir)
	if err != nil {
		t.Fatal(err)
	}

	assertUpdate(t, update, []string{"a", "c"}, []string{"b"})
	assertPaths(t, idx, filepath.Join(dir, "a.md"), filepath.Join(dir, "c.md"))
}

func TestRefreshUnchangedIsNoOp(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "a", "body")
	idx := &Index{}
	if _, err := Refresh(idx, dir); err != nil {
		t.Fatal(err)
	}

	update, err := Refresh(idx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !update.Empty() {
		t.Fatalf("second Refresh() = %+v, want empty", update)
	}
}

func TestRefreshRenamedNoteKeepsID(t *testing.T) {
	dir := t.TempDir()
	oldPath := writeNote(t, dir, "a.md", "a", "body")
	idx := &Index{}
	if _, err := Refresh(idx, dir); err != nil {
		t.Fatal(err)
	}

	newPath := filepath.Join(dir, "a-renamed.md")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	update, err := Refresh(idx, dir)
	if err != nil {
		t.Fatal(err)
	}

	assertUpdate(t, update, []string{"a"}, nil)
	assertPaths(t, idx, newPath)
}

func TestRefreshChangedIDRemovesOldID(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "a", "body")
	idx := &Index{}
	if _, err := Refresh(idx, dir); err != nil {
		t.Fatal(err)
	}

	writeNote(t, dir, "a.md", "a2", "body")
	update, err := Refresh(idx, dir)
	if err != nil {
		t.Fatal(err)
	}

	assertUpdate(t, update, []string{"a2"}, []string{"a"})
}

func TestRefreshMissingNotesDirFailsWithoutTouchingIndex(t *testing.T) {
	entries := []note.IndexEntry{{ID: "a", Path: "/gone/a.md"}}
	idx := &Index{Entries: append([]note.IndexEntry{}, entries...)}

	if _, err := Refresh(idx, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Refresh() on a missing notes dir returned no error")
	}
	if !reflect.DeepEqual(idx.Entries, entries) {
		t.Fatalf("Refresh() changed the index to %+v", idx.Entries)
	}
}

func TestRebuildForIndexesEveryNote(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "b.md", "b", "body")
	writeNote(t, dir, "a.md", "a", "body")

	idx, err := RebuildFor(dir)
	if err != nil {
		t.Fatal(err)
	}

	assertPaths(t, idx, filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md"))
}

func TestSaveForAndLoadForRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := t.TempDir()
	idx := &Index{Entries: []note.IndexEntry{{ID: "a", Path: filepath.Join(dir, "a.md"), Tags: []string{}, Links: []string{}}}}

	if err := idx.SaveFor(dir); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFor(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, idx) {
		t.Fatalf("LoadFor() = %+v, want %+v", got, idx)
	}
}

func writeNote(t *testing.T, dir, filename, id, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	n := &note.Note{ID: id, Title: id, Tags: []string{}, Links: []string{}, Created: "2026-01-01", Modified: "2026-01-01"}
	if err := os.WriteFile(path, []byte(n.ToMarkdown(content)), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertUpdate(t *testing.T, update Update, upserted, removed []string) {
	t.Helper()
	var gotUpserted []string
	for _, e := range update.Upserted {
		gotUpserted = append(gotUpserted, e.ID)
	}
	sort.Strings(gotUpserted)
	gotRemoved := append([]string{}, update.Removed...)
	sort.Strings(gotRemoved)
	if !reflect.DeepEqual(gotUpserted, upserted) {
		t.Errorf("upserted IDs = %v, want %v", gotUpserted, upserted)
	}
	if len(gotRemoved) == 0 {
		gotRemoved = nil
	}
	if !reflect.DeepEqual(gotRemoved, removed) {
		t.Errorf("removed IDs = %v, want %v", gotRemoved, removed)
	}
}

func assertPaths(t *testing.T, idx *Index, want ...string) {
	t.Helper()
	var got []string
	for _, e := range idx.Entries {
		got = append(got, e.Path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("index paths = %v, want %v", got, want)
	}
}
