package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/toshism/tnotes/internal/config"
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
		testEntry(t, dir, "required", "Required title", []string{"foo"}, "required useful text"),
		testEntry(t, dir, "excluded", "Excluded title", []string{"bar"}, "required excluded text"),
		testEntry(t, dir, "prefixed", "Prefixed note", []string{"project:foo"}, "prefixed text"),
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
			name:    "mixed phrase and word query uses AND semantics",
			query:   Query{Text: `"lander policy" exact`, Limit: 0},
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
			name:    "boolean OR over tag field alias uses exact tag values",
			query:   Query{Text: "tag:foo OR tag:bar", Limit: 0},
			wantIDs: []string{"required", "excluded"},
		},
		{
			name:    "negative tag alias with colon-containing value",
			query:   Query{Text: "+lander -tag:project:other", Limit: 0},
			wantIDs: []string{"landing", "policy", "phrase"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchWithIndexPath(idx, tt.query, indexPath)
			if err != nil {
				t.Fatal(err)
			}
			gotIDs := resultIDs(got)
			if tt.name == "multi-word query uses word AND semantics" || tt.name == "single word hits stemmed variants" || tt.name == "boolean OR over tag field alias uses exact tag values" || tt.name == "negative tag alias with colon-containing value" {
				assertSameIDs(t, gotIDs, tt.wantIDs)
				return
			}
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Fatalf("Search() IDs = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestCodeIdentifierAnalyzer(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "snake", "Snake fixture", []string{"project:teamtosh"}, "The rollout mentions teamtosh_lander and nothing else."),
		testEntry(t, dir, "snake-team", "Snake team fixture", nil, "This note mentions teamtosh only."),
		testEntry(t, dir, "snake-land", "Snake land fixture", nil, "This note mentions lander only."),
		testEntry(t, dir, "camel", "Camel fixture", nil, "React state flows through WaffleFlagsContext."),
		testEntry(t, dir, "camel-context", "Camel context fixture", nil, "This note mentions context only."),
		testEntry(t, dir, "mixed", "Mixed fixture", nil, "The getUserById_helper path validates identifier splitting."),
		testEntry(t, dir, "stem", "Stem fixture", nil, "The learnings_agent writes durable notes."),
		testEntry(t, dir, "tag-only", "Tag-only fixture", []string{"project:teamtosh"}, "No matching body tokens here."),
		testEntry(t, dir, "tag-suffix", "Tag suffix fixture", []string{"project:teamtosh-anything-else"}, "No matching body tokens here either."),
	}}
	indexPath := filepath.Join(dir, ".tnotes", "bleve")

	tests := []struct {
		name    string
		query   Query
		wantIDs []string
	}{
		{name: "snake case matches first part", query: Query{Text: "teamtosh", Limit: 0}, wantIDs: []string{"snake", "snake-team"}},
		{name: "snake case matches second part", query: Query{Text: "lander", Limit: 0}, wantIDs: []string{"snake", "snake-land"}},
		{name: "snake case exact token still matches without component overmatch", query: Query{Text: "teamtosh_lander", Limit: 0}, wantIDs: []string{"snake"}},
		{name: "camel case matches first part", query: Query{Text: "waffle", Limit: 0}, wantIDs: []string{"camel"}},
		{name: "camel case matches middle part", query: Query{Text: "flags", Limit: 0}, wantIDs: []string{"camel"}},
		{name: "camel case matches last part", query: Query{Text: "context", Limit: 0}, wantIDs: []string{"camel", "camel-context"}},
		{name: "camel case exact token still matches without component overmatch", query: Query{Text: "WaffleFlagsContext", Limit: 0}, wantIDs: []string{"camel"}},
		{name: "mixed identifier matches get", query: Query{Text: "get", Limit: 0}, wantIDs: []string{"mixed"}},
		{name: "mixed identifier matches user", query: Query{Text: "user", Limit: 0}, wantIDs: []string{"mixed"}},
		{name: "mixed identifier matches by", query: Query{Text: "by", Limit: 0}, wantIDs: []string{"mixed"}},
		{name: "mixed identifier matches id", query: Query{Text: "id", Limit: 0}, wantIDs: []string{"mixed"}},
		{name: "mixed identifier matches helper", query: Query{Text: "helper", Limit: 0}, wantIDs: []string{"mixed"}},
		{name: "post split stemming matches learn", query: Query{Text: "learn", Limit: 0}, wantIDs: []string{"stem"}},
		{name: "post split stemming matches learning", query: Query{Text: "learning", Limit: 0}, wantIDs: []string{"stem"}},
		{name: "post split stemming matches agent", query: Query{Text: "agent", Limit: 0}, wantIDs: []string{"stem"}},
		{name: "exact tag filter does not match tag suffix", query: Query{Tags: []string{"project:teamtosh"}, Limit: 0}, wantIDs: []string{"snake", "tag-only"}},
		{name: "tag values are not searched as text", query: Query{Text: "project", Limit: 0}, wantIDs: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchWithIndexPath(idx, tt.query, indexPath)
			if err != nil {
				t.Fatal(err)
			}
			assertSameIDs(t, resultIDs(got), tt.wantIDs)
		})
	}
}

func TestTitleOnlyStemmedVariantMatches(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "title-only", "Lander", nil, "no matching body text"),
	}}

	got, err := SearchWithIndexPath(idx, Query{Text: "land"}, filepath.Join(dir, ".tnotes", "bleve"))
	if err != nil {
		t.Fatal(err)
	}
	if gotIDs := resultIDs(got); !reflect.DeepEqual(gotIDs, []string{"title-only"}) {
		t.Fatalf("Search() IDs = %v", gotIDs)
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

func TestNoteIDQuery(t *testing.T) {
	dir := t.TempDir()
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "20260411002302", "In-flight tracker", nil, "board body"),
		testEntry(t, dir, "20260411093000", "Same day note", nil, "unrelated body"),
		testEntry(t, dir, "20260412080000", "Next day note", nil, "unrelated body"),
		testEntry(t, dir, "20260916030403", "Follow-up to 20260411002302", nil, strings.Repeat("see 20260411002302 ", 10)),
	}}
	indexPath := filepath.Join(dir, ".tnotes", "bleve")

	search := func(text string) []string {
		t.Helper()
		got, err := SearchWithIndexPath(idx, Query{Text: text, Limit: 0}, indexPath)
		if err != nil {
			t.Fatal(err)
		}
		return resultIDs(got)
	}

	t.Run("full ID ranks the note itself above notes that mention it", func(t *testing.T) {
		want := []string{"20260411002302", "20260916030403"}
		if got := search("20260411002302"); !reflect.DeepEqual(got, want) {
			t.Fatalf("Search() IDs = %v, want %v", got, want)
		}
	})

	t.Run("date prefix matches notes created that day", func(t *testing.T) {
		assertSameIDs(t, search("20260411"), []string{"20260411002302", "20260411093000"})
	})

	t.Run("date prefix combines with other words", func(t *testing.T) {
		assertSameIDs(t, search("20260411 tracker"), []string{"20260411002302"})
	})

	t.Run("prefix shorter than a date does not match IDs", func(t *testing.T) {
		assertSameIDs(t, search("2026041"), nil)
	})
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

func TestIndexEntryWithIndexBuildsFullCorpusWhenBleveMissing(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, ".tnotes", "bleve")
	idx := &index.Index{Entries: []note.IndexEntry{
		testEntry(t, dir, "existing", "Existing", nil, "old corpus needle"),
	}}

	entry := testEntry(t, dir, "added", "Added", nil, "fresh body")
	idx.AddEntry(entry)
	if err := IndexEntryWithIndexAt(idx, entry, indexPath); err != nil {
		t.Fatal(err)
	}

	got, err := SearchWithIndexPath(idx, Query{Text: "needle"}, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotIDs := resultIDs(got); !reflect.DeepEqual(gotIDs, []string{"existing"}) {
		t.Fatalf("Search() IDs = %v", gotIDs)
	}
}

func TestLoadFreshFindsNotesAddedEditedAndDeletedOnDisk(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := t.TempDir()
	testEntry(t, dir, "a", "A", nil, "alpha body")
	testEntry(t, dir, "b", "B", nil, "bravo body")
	assertFreshSearch(t, dir, "alpha", "a")

	testEntry(t, dir, "a", "A", nil, "zulu body, edited")
	testEntry(t, dir, "c", "C", nil, "charlie body")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}

	assertFreshSearch(t, dir, "zulu", "a")
	assertFreshSearch(t, dir, "alpha")
	assertFreshSearch(t, dir, "charlie", "c")
	assertFreshSearch(t, dir, "bravo")
}

func TestLoadFreshDoesNotRewriteUnchangedIndex(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := t.TempDir()
	testEntry(t, dir, "a", "A", nil, "alpha body")
	if _, err := LoadFresh(dir); err != nil {
		t.Fatal(err)
	}
	indexFile := config.IndexFileFor(dir)
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(indexFile, past, past); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadFresh(dir); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(indexFile)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("index.json rewritten at %v with no note changes", info.ModTime())
	}
}

func TestLoadFreshToleratesNoteWithoutFrontmatter(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := t.TempDir()
	testEntry(t, dir, "a", "A", nil, "alpha body")
	assertFreshSearch(t, dir, "alpha", "a")

	if err := os.WriteFile(filepath.Join(dir, "plain.md"), []byte("no frontmatter here\n"), 0644); err != nil {
		t.Fatal(err)
	}

	assertFreshSearch(t, dir, "alpha", "a")
}

// One synced notes folder, mounted at a different path on each machine.
func TestLoadFreshReturnsPathsUnderEachMachinesMount(t *testing.T) {
	mountA := t.TempDir()
	mountB := t.TempDir()
	cacheA := t.TempDir()
	cacheB := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", cacheA)
	testEntry(t, mountA, "a", "A", nil, "alpha body")
	assertFreshSearch(t, mountA, "alpha", "a")
	syncFolder(t, mountA, mountB)

	t.Setenv("XDG_CACHE_HOME", cacheB)
	assertFreshSearchPaths(t, mountB, "alpha", filepath.Join(mountB, "a.md"))

	t.Setenv("XDG_CACHE_HOME", cacheA)
	testEntry(t, mountA, "b", "B", nil, "bravo body")
	assertFreshSearch(t, mountA, "bravo", "b")
	syncFolder(t, mountA, mountB)

	t.Setenv("XDG_CACHE_HOME", cacheB)
	assertFreshSearchPaths(t, mountB, "bravo", filepath.Join(mountB, "b.md"))
}

// syncFolder copies every file from src to dst with its modification time, as
// Syncthing does.
func syncFolder(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(dst, e.Name())
		if err := os.WriteFile(target, data, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(target, info.ModTime(), info.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
}

func freshSearch(t *testing.T, notesDir, text string) []Result {
	t.Helper()
	idx, err := LoadFresh(notesDir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := SearchWithIndexPath(idx, Query{Text: text}, config.BleveIndexDirFor(notesDir))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func assertFreshSearch(t *testing.T, notesDir, text string, wantIDs ...string) {
	t.Helper()
	gotIDs := resultIDs(freshSearch(t, notesDir, text))
	if len(gotIDs) == 0 && len(wantIDs) == 0 {
		return
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("search %q IDs = %v, want %v", text, gotIDs, wantIDs)
	}
}

func assertFreshSearchPaths(t *testing.T, notesDir, text string, wantPaths ...string) {
	t.Helper()
	var gotPaths []string
	for _, r := range freshSearch(t, notesDir, text) {
		gotPaths = append(gotPaths, r.Entry.Path)
	}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("search %q paths = %v, want %v", text, gotPaths, wantPaths)
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
	if strings.Contains(string(data), `"score"`) {
		t.Fatalf("Result JSON should not include score: %s", data)
	}
}
