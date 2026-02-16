package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFromDotfile(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, ".tnotes-project"), []byte("  my-project  \n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	got := resolveFromDotfile(dir)
	if got != "my-project" {
		t.Errorf("resolveFromDotfile(%q) = %q, want %q", dir, got, "my-project")
	}
}

func TestResolveFromDotfileWalksParents(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "deep")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".tnotes-project"), []byte("parent-project\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := resolveFromDotfile(child)
	if got != "parent-project" {
		t.Errorf("resolveFromDotfile(%q) = %q, want %q", child, got, "parent-project")
	}
}

func TestResolveFromDotfileReturnsEmptyWhenMissing(t *testing.T) {
	dir := t.TempDir()

	got := resolveFromDotfile(dir)
	if got != "" {
		t.Errorf("resolveFromDotfile(%q) = %q, want empty string", dir, got)
	}
}

func TestNormalizeGitRemote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:acme/platform.git", "acme/platform"},
		{"https://github.com/acme/platform.git", "acme/platform"},
		{"https://github.com/acme/platform", "acme/platform"},
		{"git@gitlab.com:org/sub/repo.git", "org/sub/repo"},
	}

	for _, tt := range tests {
		got := normalizeGitRemote(tt.input)
		if got != tt.want {
			t.Errorf("normalizeGitRemote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveInfo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".tnotes-project"), []byte("integration-project\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := Resolve(dir)

	if info.Project != "integration-project" {
		t.Errorf("Resolve(%q).Project = %q, want %q", dir, info.Project, "integration-project")
	}
	if info.Path != dir {
		t.Errorf("Resolve(%q).Path = %q, want %q", dir, info.Path, dir)
	}
}

func TestResolveFallsBackToDirBasename(t *testing.T) {
	// Create a temp dir with a known basename
	parent := t.TempDir()
	dir := filepath.Join(parent, "my-cool-project")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	info := Resolve(dir)

	if info.Project != "my-cool-project" {
		t.Errorf("Resolve(%q).Project = %q, want %q", dir, info.Project, "my-cool-project")
	}
	if info.Path != dir {
		t.Errorf("Resolve(%q).Path = %q, want %q", dir, info.Path, dir)
	}
}
