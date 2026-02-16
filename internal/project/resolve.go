package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds the resolved project identity for a directory.
type Info struct {
	Project string
	Path    string
}

// Resolve returns project identity for the given directory.
// Resolution order: .tnotes-project dotfile, git remote origin, directory basename.
func Resolve(dir string) Info {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	project := resolveFromDotfile(absDir)
	if project == "" {
		project = resolveFromGit(absDir)
	}
	if project == "" {
		project = filepath.Base(absDir)
	}

	return Info{
		Project: project,
		Path:    absDir,
	}
}

// resolveFromDotfile walks from dir up to the filesystem root looking for
// a .tnotes-project file. Returns the trimmed contents or empty string.
func resolveFromDotfile(dir string) string {
	current := dir
	for {
		data, err := os.ReadFile(filepath.Join(current, ".tnotes-project"))
		if err == nil {
			name := strings.TrimSpace(string(data))
			if name != "" {
				return name
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// resolveFromGit runs git remote get-url origin and normalizes the result.
func resolveFromGit(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return normalizeGitRemote(strings.TrimSpace(string(out)))
}

// normalizeGitRemote converts a git remote URL to org/repo format.
func normalizeGitRemote(raw string) string {
	// SSH: git@host:path.git
	if strings.HasPrefix(raw, "git@") {
		idx := strings.Index(raw, ":")
		if idx == -1 {
			return raw
		}
		path := raw[idx+1:]
		path = strings.TrimSuffix(path, ".git")
		return path
	}

	// HTTPS: https://host/path.git
	raw = strings.TrimSuffix(raw, ".git")
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(raw, prefix) {
			raw = raw[len(prefix):]
			break
		}
	}
	// Strip the hostname (first segment)
	idx := strings.Index(raw, "/")
	if idx == -1 {
		return raw
	}
	return raw[idx+1:]
}
