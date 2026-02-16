package config

import (
	"os"
	"path/filepath"
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
