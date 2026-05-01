package search

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/note"
)

func TestSearch(t *testing.T) {
	idx := testIndex(t)

	tests := []struct {
		name        string
		query       Query
		wantIDs     []string
		wantMatches map[string][]string
	}{
		{
			name:    "tag-only returns all entries matching tag",
			query:   Query{Tags: []string{"project:TeamTosh"}},
			wantIDs: []string{"team-title", "team-content", "team-tag-only"},
			wantMatches: map[string][]string{
				"team-title":    {"tag"},
				"team-content":  {"tag"},
				"team-tag-only": {"tag"},
			},
		},
		{
			name:    "text-only returns title or content matches",
			query:   Query{Text: "shared text needle"},
			wantIDs: []string{"team-title", "team-content"},
			wantMatches: map[string][]string{
				"team-title":   {"title"},
				"team-content": {"content"},
			},
		},
		{
			name:    "tag and text returns entry when both match",
			query:   Query{Text: "shared text needle title", Tags: []string{"project:TeamTosh"}},
			wantIDs: []string{"team-title"},
			wantMatches: map[string][]string{
				"team-title": {"tag", "title"},
			},
		},
		{
			name:        "tag and text returns nothing when only tag matches",
			query:       Query{Text: "nonexistent gibberish", Tags: []string{"project:TeamTosh"}},
			wantIDs:     []string{},
			wantMatches: map[string][]string{},
		},
		{
			name:        "tag and text returns nothing when only text matches",
			query:       Query{Text: "other tag needle", Tags: []string{"project:TeamTosh"}},
			wantIDs:     []string{},
			wantMatches: map[string][]string{},
		},
		{
			name:    "no filters returns everything",
			query:   Query{},
			wantIDs: []string{"team-title", "team-content", "team-tag-only", "other-text", "case-entry"},
			wantMatches: map[string][]string{
				"team-title":    {"all"},
				"team-content":  {"all"},
				"team-tag-only": {"all"},
				"other-text":    {"all"},
				"case-entry":    {"all"},
			},
		},
		{
			name:    "tag and text matching is case-insensitive",
			query:   Query{Text: "mixed case text", Tags: []string{"project:case"}},
			wantIDs: []string{"case-entry"},
			wantMatches: map[string][]string{
				"case-entry": {"tag", "title"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Search(idx, tt.query)
			if err != nil {
				t.Fatal(err)
			}

			if gotIDs := resultIDs(got); !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("Search() IDs = %v, want %v", gotIDs, tt.wantIDs)
			}

			gotMatches := resultMatches(got)
			if !reflect.DeepEqual(gotMatches, tt.wantMatches) {
				t.Errorf("Search() matches = %v, want %v", gotMatches, tt.wantMatches)
			}
		})
	}
}

func testIndex(t *testing.T) *index.Index {
	t.Helper()

	dir := t.TempDir()
	entries := []note.IndexEntry{
		testEntry(t, dir, "team-title", "Shared Text Needle Title", []string{"project:TeamTosh"}, "no relevant body"),
		testEntry(t, dir, "team-content", "Content Note", []string{"project:TeamTosh"}, "body has shared text needle"),
		testEntry(t, dir, "team-tag-only", "Plain Team Note", []string{"project:TeamTosh"}, "plain body"),
		testEntry(t, dir, "other-text", "Other Tag Needle", []string{"project:Other"}, "plain body"),
		testEntry(t, dir, "case-entry", "Mixed Case Text", []string{"Project:Case"}, "plain body"),
	}

	return &index.Index{Entries: entries}
}

func testEntry(t *testing.T, dir, id, title string, tags []string, content string) note.IndexEntry {
	t.Helper()

	path := filepath.Join(dir, id+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return note.IndexEntry{
		ID:    id,
		Title: title,
		Tags:  tags,
		Path:  path,
	}
}

func resultIDs(results []Result) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.Entry.ID)
	}
	return ids
}

func resultMatches(results []Result) map[string][]string {
	matches := make(map[string][]string, len(results))
	for _, result := range results {
		matches[result.Entry.ID] = result.Matches
	}
	return matches
}
