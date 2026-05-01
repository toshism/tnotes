package search

import (
	"os"
	"strings"

	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/note"
)

// Query represents search parameters
type Query struct {
	Text string   // Full-text search (title + content)
	Tags []string // Filter by tags (exact match)
}

// Result represents a search result with relevance info
type Result struct {
	Entry   note.IndexEntry `json:"entry"`
	Matches []string        `json:"matches,omitempty"` // What matched (title, tag, content)
}

// Search performs a search against the index
func Search(idx *index.Index, q Query) ([]Result, error) {
	var results []Result

	for _, entry := range idx.Entries {
		matches := matchEntry(entry, q)
		if len(matches) > 0 {
			results = append(results, Result{
				Entry:   entry,
				Matches: matches,
			})
		}
	}

	return results, nil
}

// matchEntry checks if an entry matches the query
func matchEntry(entry note.IndexEntry, q Query) []string {
	var matches []string

	// Tag filter (if specified, all tags must match)
	if len(q.Tags) > 0 {
		tagSet := make(map[string]bool)
		for _, t := range entry.Tags {
			tagSet[strings.ToLower(t)] = true
		}

		for _, qt := range q.Tags {
			if !tagSet[strings.ToLower(qt)] {
				return nil // Tag not found, no match
			}
		}
		matches = append(matches, "tag")
	}

	// Text search
	if q.Text != "" {
		textLower := strings.ToLower(q.Text)
		titleMatch := strings.Contains(strings.ToLower(entry.Title), textLower)
		contentMatch := contentMatches(entry.Path, textLower)

		// Search in title
		if titleMatch {
			matches = append(matches, "title")
		}

		// Search in content (read file)
		if contentMatch {
			matches = append(matches, "content")
		}

		// If text was specified but neither title nor content matched, return nil
		if !titleMatch && !contentMatch {
			return nil
		}
	}

	// If no filters specified, match everything
	if q.Text == "" && len(q.Tags) == 0 {
		return []string{"all"}
	}

	return matches
}

// contentMatches checks if file content contains the search text
func contentMatches(path, searchText string) bool {
	if path == "" {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	content := strings.ToLower(string(data))
	return strings.Contains(content, searchText)
}
