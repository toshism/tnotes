package note

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Note represents a single note with its metadata and content
type Note struct {
	ID       string   `json:"id" yaml:"id"`
	Title    string   `json:"title" yaml:"title"`
	Tags     []string `json:"tags" yaml:"tags"`
	Links    []string `json:"links" yaml:"links"`
	Created  string   `json:"created" yaml:"created"`
	Modified string   `json:"modified" yaml:"modified"`
	Path     string   `json:"path" yaml:"-"`
}

// IndexEntry is a lightweight version of Note for the index
type IndexEntry struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Links    []string `json:"links"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
	Path     string   `json:"path"`
	ModTime  int64    `json:"mod_time"` // Unix timestamp for cache invalidation
}

// GenerateID creates a new timestamp-based ID
func GenerateID() string {
	return time.Now().Format("20060102150405")
}

// Today returns today's date in YYYY-MM-DD format
func Today() string {
	return time.Now().Format("2006-01-02")
}

// NewNote creates a new note with the given title and tags
func NewNote(title string, tags []string) *Note {
	now := Today()
	return &Note{
		ID:       GenerateID(),
		Title:    title,
		Tags:     tags,
		Links:    []string{},
		Created:  now,
		Modified: now,
	}
}

// ToMarkdown serializes the note to markdown with frontmatter
func (n *Note) ToMarkdown(content string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %q\n", n.ID))
	sb.WriteString(fmt.Sprintf("title: %q\n", n.Title))

	// Tags as YAML inline array
	sb.WriteString("tags: [")
	for i, t := range n.Tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q", t))
	}
	sb.WriteString("]\n")

	// Links as YAML inline array
	sb.WriteString("links: [")
	for i, l := range n.Links {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q", l))
	}
	sb.WriteString("]\n")

	sb.WriteString(fmt.Sprintf("created: %q\n", n.Created))
	sb.WriteString(fmt.Sprintf("modified: %q\n", n.Modified))
	sb.WriteString("---\n\n")

	// Only add heading if content doesn't already start with one
	if !strings.HasPrefix(content, "# ") {
		sb.WriteString(fmt.Sprintf("# %s\n\n", n.Title))
	}

	if content != "" {
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	return sb.String()
}

// ParseFile reads a markdown file and parses its frontmatter
func ParseFile(path string) (*Note, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var frontmatter strings.Builder
	var body strings.Builder
	scanner := bufio.NewScanner(file)

	// States: 0 = before frontmatter, 1 = in frontmatter, 2 = body
	state := 0
	dashCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		if state == 0 && line == "---" {
			state = 1
			dashCount++
			continue
		}

		if state == 1 {
			if line == "---" {
				state = 2
				continue
			}
			frontmatter.WriteString(line)
			frontmatter.WriteString("\n")
			continue
		}

		if state == 2 {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	// Parse YAML frontmatter
	var note Note
	if err := yaml.Unmarshal([]byte(frontmatter.String()), &note); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	note.Path = path

	return &note, strings.TrimSpace(body.String()), nil
}

// ToIndexEntry converts a Note to an IndexEntry
func (n *Note) ToIndexEntry() IndexEntry {
	modTime := int64(0)
	if n.Path != "" {
		if info, err := os.Stat(n.Path); err == nil {
			modTime = info.ModTime().Unix()
		}
	}

	return IndexEntry{
		ID:       n.ID,
		Title:    n.Title,
		Tags:     n.Tags,
		Links:    n.Links,
		Created:  n.Created,
		Modified: n.Modified,
		Path:     n.Path,
		ModTime:  modTime,
	}
}

// Filename returns the suggested filename for this note
func (n *Note) Filename() string {
	// Sanitize title for filename
	safe := strings.Map(func(r rune) rune {
		if r == ' ' {
			return '-'
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, n.Title)

	safe = strings.ToLower(safe)
	if len(safe) > 50 {
		safe = safe[:50]
	}

	return fmt.Sprintf("%s-%s.md", n.ID, safe)
}

// FindNoteByID searches for a note by ID in the given directory
func FindNoteByID(dir, id string) (*Note, string, error) {
	var foundNote *Note
	var foundContent string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip .tnotes directory
		if info.IsDir() && info.Name() == ".tnotes" {
			return filepath.SkipDir
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		note, content, err := ParseFile(path)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		if note.ID == id {
			foundNote = note
			foundContent = content
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, "", err
	}

	if foundNote == nil {
		return nil, "", fmt.Errorf("note with ID %s not found", id)
	}

	return foundNote, foundContent, nil
}
