package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultNotesDirIsTnotes(t *testing.T) {
	d := defaultNotesDir()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "tnotes")
	if d != expected {
		t.Errorf("expected %s, got %s", expected, d)
	}
}

func TestInitWithEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TNOTES_DIR", dir)
	Init("", "")
	if NotesDir != dir {
		t.Errorf("expected %s, got %s", dir, NotesDir)
	}
}

func TestInitWithExplicitDir(t *testing.T) {
	dir := t.TempDir()
	Init("", dir)
	if NotesDir != dir {
		t.Errorf("expected %s, got %s", dir, NotesDir)
	}
}

func TestIndexDirIsInCacheOutsideNotesDir(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	notes := t.TempDir()

	got := IndexDirFor(notes)
	if !strings.HasPrefix(got, filepath.Join(cache, "tnotes")+string(filepath.Separator)) {
		t.Errorf("IndexDirFor(%s) = %s, want under %s", notes, got, filepath.Join(cache, "tnotes"))
	}
	if strings.HasPrefix(got, notes) {
		t.Errorf("IndexDirFor(%s) = %s, want outside the notes dir", notes, got)
	}
}

func TestIndexDirIsSharedBySymlinkAndTarget(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "notes")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if a, b := IndexDirFor(target), IndexDirFor(link); a != b {
		t.Errorf("IndexDirFor(target) = %s, IndexDirFor(link) = %s, want equal", a, b)
	}
}

func TestIndexDirDiffersPerNotesDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	if a, b := IndexDirFor(t.TempDir()), IndexDirFor(t.TempDir()); a == b {
		t.Errorf("two notes dirs share index dir %s", a)
	}
}
