package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/note"
)

func TestBleveSearch(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "landing", "Moon landing policy", []string{"project:space", "topic:mission"}, "The lander touched down safely during the mission."),
		testEntry(t, dir, "policy", "Policy background", []string{"project:space"}, "A moon lander needs a careful operations plan."),
		testEntry(t, dir, "phrase", "Phrase note", []string{"project:space"}, "This note contains the exact lander policy phrase."),
		testEntry(t, dir, "other", "Other note", []string{"project:other", "topic:archive"}, "The mission excluded this policy."),
		testEntry(t, dir, "required", "Required title", []string{"tag:foo"}, "required useful text"),
		testEntry(t, dir, "excluded", "Excluded title", []string{"tag:bar"}, "required excluded text"),
	}}
	indexPath := filepath.Join(dir, ".tnotes", "bleve")

	tests := []struct {
		name    string
		query   Query
		wantIDs []string
	}{
		{
			name:    "single word hits stemmed variants",
			query:   Query{Text: "land", Limit: 0},
			wantIDs: []string{"landing", "policy", "phrase"},
		},
		{
			name:    "multi-word query uses word AND semantics",
			query:   Query{Text: "lander policy", Limit: 0},
			wantIDs: []string{"landing", "policy", "phrase"},
		},
		{
			name:    "phrase query requires exact phrase",
			query:   Query{Text: `"lander policy"`, Limit: 0},
			wantIDs: []string{"phrase"},
		},
		{
			name:    "tag filter alone returns all matching notes",
			query:   Query{Tags: []string{"project:space"}, Limit: 0},
			wantIDs: []string{"landing", "policy", "phrase"},
		},
		{
			name:    "tag and text use AND semantics",
			query:   Query{Text: "mission", Tags: []string{"project:space"}, Limit: 0},
			wantIDs: []string{"landing"},
		},
		{
			name:    "boolean required and excluded operators",
			query:   Query{Text: "+required -excluded", Limit: 0},
			wantIDs: []string{"required"},
		},
		{
			name:    "boolean OR over tag field alias",
			query:   Query{Text: "tag:foo OR tag:bar", Limit: 0},
			wantIDs: []string{"required", "excluded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchWithIndexPath(idx, tt.query, indexPath)
			if err != nil {
				t.Fatal(err)
			}
			gotIDs := resultIDs(got)
			if tt.name == "multi-word query uses word AND semantics" || tt.name == "single word hits stemmed variants" || tt.name == "boolean OR over tag field alias" {
				assertSameIDs(t, gotIDs, tt.wantIDs)
				return
			}
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Fatalf("Search() IDs = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestRankingTitleMatchFirst(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "body", "Generic note", nil, strings.Repeat("ranking ", 20)),
		testEntry(t, dir, "title", "Ranking", nil, "short body"),
	}}

	got, err := SearchWithIndexPath(idx, Query{Text: "ranking", Limit: 0}, filepath.Join(dir, ".tnotes", "bleve"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 || got[0].Entry.ID != "title" {
		t.Fatalf("title match should rank first, got %v", resultIDs(got))
	}
}

func TestSnippets(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "snippet", "Snippet note", nil, "before important context after"),
	}}

	got, err := SearchWithIndexPath(idx, Query{Text: "important", Snippets: true}, filepath.Join(dir, ".tnotes", "bleve"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Snippet, "important") {
		t.Fatalf("snippet = %#v", got)
	}
}

func TestMigrationAutoBuildsMissingBleveIndex(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "migration", "Migration", nil, "autobuild needle"),
	}}
	indexPath := filepath.Join(dir, ".tnotes", "bleve")

	got, err := SearchWithIndexPath(idx, Query{Text: "needle"}, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("bleve index was not created: %v", err)
	}
	if gotIDs := resultIDs(got); !reflect.DeepEqual(gotIDs, []string{"migration"}) {
		t.Fatalf("Search() IDs = %v", gotIDs)
	}
}

func TestIndexEntryUpdatesBleve(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, ".tnotes", "bleve")
	idx := &index.Index{Entries: []note.IndexEntry{}}
	if err := RebuildBleveAt(idx, indexPath); err != nil {
		t.Fatal(err)
	}

	entry := testEntry(t, dir, "added", "Added", nil, "fresh searchable body")
	idx.AddEntry(entry)
	if err := IndexEntryAt(entry, indexPath); err != nil {
		t.Fatal(err)
	}

	got, err := SearchWithIndexPath(idx, Query{Text: "fresh"}, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotIDs := resultIDs(got); !reflect.DeepEqual(gotIDs, []string{"added"}) {
		t.Fatalf("Search() IDs = %v", gotIDs)
	}
}

func testEntry(t *testing.T, dir, id, title string, tags []string, content string) note.IndexEntry {
	t.Helper()
	path := filepath.Join(dir, id+".md")
	n := &note.Note{ID: id, Title: title, Tags: tags, Links: []string{}, Created: "2026-01-01", Modified: "2026-01-01"}
	if err := os.WriteFile(path, []byte(n.ToMarkdown(content)), 0644); err != nil {
		t.Fatal(err)
	}
	n.Path = path
	return n.ToIndexEntry()
}

func assertSameIDs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Search() IDs = %v, want same set as %v", got, want)
	}
	seen := map[string]int{}
	for _, id := range got {
		seen[id]++
	}
	for _, id := range want {
		seen[id]--
	}
	for id, count := range seen {
		if count != 0 {
			t.Fatalf("Search() IDs = %v, want same set as %v (mismatch %s)", got, want, id)
		}
	}
}

func resultIDs(results []Result) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.Entry.ID)
	}
	return ids
}

func TestResultShapeKeepsEntry(t *testing.T) {
	data, err := json.Marshal(Result{Entry: note.IndexEntry{ID: "id"}, Score: 1, Snippet: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"entry"`) {
		t.Fatalf("Result JSON missing entry: %s", data)
	}
}
