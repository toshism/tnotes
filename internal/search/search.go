package search

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	blevesearch "github.com/blevesearch/bleve/v2/search"
	blevequery "github.com/blevesearch/bleve/v2/search/query"
	"github.com/toshism/tnotes/internal/config"
	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/note"
)

const defaultLimit = 20

// Query represents search parameters
type Query struct {
	Text     string   // Full-text search (title + content)
	Tags     []string // Filter by tags (exact match)
	Limit    int      // Maximum results; 0 means unlimited
	Snippets bool     // Include highlighted matching snippets
}

// Result represents a search result with relevance info
type Result struct {
	Entry   note.IndexEntry `json:"entry"`
	Matches []string        `json:"matches,omitempty"` // What matched (title, tag, content)
	Score   float64         `json:"score,omitempty"`
	Snippet string          `json:"snippet,omitempty"`
}

type bleveDocument struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
}

// Search performs a search against the configured bleve index, auto-building it
// from index.json metadata if this is the first search after upgrade.
func Search(idx *index.Index, q Query) ([]Result, error) {
	return SearchWithIndexPath(idx, q, config.BleveIndexDir())
}

// SearchWithIndexPath performs a search against a specific bleve index path.
func SearchWithIndexPath(idx *index.Index, q Query, indexPath string) ([]Result, error) {
	if q.Limit < 0 {
		q.Limit = defaultLimit
	}
	if strings.TrimSpace(q.Text) == "" {
		return metadataOnlySearch(idx, q), nil
	}

	b, err := ensureBleveIndex(idx, indexPath)
	if err != nil {
		return nil, err
	}
	defer b.Close()

	bleveQ := buildBleveQuery(q)
	size := q.Limit
	if size == 0 {
		size = len(idx.Entries)
	}
	if size == 0 {
		size = 10
	}

	req := bleve.NewSearchRequestOptions(bleveQ, size, 0, false)
	req.IncludeLocations = true
	if q.Snippets && q.Text != "" {
		h := bleve.NewHighlight()
		h.AddField("title")
		h.AddField("content")
		req.Highlight = h
	}

	resp, err := b.Search(req)
	if err != nil {
		return nil, err
	}

	entries := make(map[string]note.IndexEntry, len(idx.Entries))
	for _, entry := range idx.Entries {
		entries[entry.ID] = entry
	}

	results := make([]Result, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		entry, ok := entries[hit.ID]
		if !ok {
			continue
		}
		results = append(results, Result{
			Entry:   entry,
			Matches: matchesFromLocations(hit.Locations, q),
			Score:   hit.Score,
			Snippet: snippetFromFragments(hit.Fragments),
		})
	}

	return results, nil
}

// RebuildBleve removes and recreates the bleve index from index.json entries.
func RebuildBleve(idx *index.Index) error {
	return RebuildBleveAt(idx, config.BleveIndexDir())
}

// RebuildBleveAt removes and recreates the bleve index at a specific path.
func RebuildBleveAt(idx *index.Index, indexPath string) error {
	if err := os.RemoveAll(indexPath); err != nil {
		return err
	}
	b, err := createBleveIndex(indexPath)
	if err != nil {
		return err
	}
	defer b.Close()
	return indexEntries(b, idx)
}

// IndexEntry upserts one entry into the configured bleve index.
func IndexEntry(entry note.IndexEntry) error {
	return IndexEntryAt(entry, config.BleveIndexDir())
}

// IndexEntryAt upserts one entry into a specific bleve index path.
func IndexEntryAt(entry note.IndexEntry, indexPath string) error {
	b, err := openOrCreateBleveIndex(indexPath)
	if err != nil {
		return err
	}
	defer b.Close()
	return b.Index(entry.ID, documentForEntry(entry))
}

func metadataOnlySearch(idx *index.Index, q Query) []Result {
	results := make([]Result, 0, len(idx.Entries))
	for _, entry := range idx.Entries {
		if !entryHasTags(entry, q.Tags) {
			continue
		}
		matches := []string{"all"}
		if len(q.Tags) > 0 {
			matches = []string{"tag"}
		}
		results = append(results, Result{Entry: entry, Matches: matches})
		if q.Limit > 0 && len(results) >= q.Limit {
			break
		}
	}
	return results
}

func entryHasTags(entry note.IndexEntry, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	tagSet := make(map[string]bool, len(entry.Tags))
	for _, tag := range entry.Tags {
		tagSet[strings.ToLower(tag)] = true
	}
	for _, tag := range tags {
		if !tagSet[strings.ToLower(strings.TrimSpace(tag))] {
			return false
		}
	}
	return true
}

func ensureBleveIndex(idx *index.Index, indexPath string) (bleve.Index, error) {
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		if err := RebuildBleveAt(idx, indexPath); err != nil {
			return nil, err
		}
	}
	return openOrCreateBleveIndex(indexPath)
}

func openOrCreateBleveIndex(indexPath string) (bleve.Index, error) {
	b, err := bleve.Open(indexPath)
	if err == nil {
		return b, nil
	}
	if _, statErr := os.Stat(indexPath); statErr == nil {
		return nil, err
	}
	return createBleveIndex(indexPath)
}

func createBleveIndex(indexPath string) (bleve.Index, error) {
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return nil, err
	}
	return bleve.New(indexPath, newIndexMapping())
}

func newIndexMapping() mapping.IndexMapping {
	idxMapping := bleve.NewIndexMapping()
	idxMapping.DefaultAnalyzer = "en"

	docMapping := bleve.NewDocumentMapping()
	textField := func() *mapping.FieldMapping {
		fm := bleve.NewTextFieldMapping()
		fm.Analyzer = "en"
		fm.Store = true
		fm.IncludeTermVectors = true
		fm.IncludeInAll = true
		return fm
	}

	keywordField := func() *mapping.FieldMapping {
		fm := bleve.NewKeywordFieldMapping()
		fm.Store = true
		fm.IncludeInAll = false
		return fm
	}

	docMapping.AddFieldMappingsAt("id", keywordField())
	docMapping.AddFieldMappingsAt("title", textField())
	docMapping.AddFieldMappingsAt("content", textField())
	docMapping.AddFieldMappingsAt("tags", keywordField())

	dateField := bleve.NewDateTimeFieldMapping()
	dateField.Store = true
	dateField.IncludeInAll = false
	docMapping.AddFieldMappingsAt("created", dateField)
	docMapping.AddFieldMappingsAt("modified", dateField)

	idxMapping.DefaultMapping = docMapping
	return idxMapping
}

func indexEntries(b bleve.Index, idx *index.Index) error {
	batch := b.NewBatch()
	for _, entry := range idx.Entries {
		batch.Index(entry.ID, documentForEntry(entry))
	}
	return b.Batch(batch)
}

func documentForEntry(entry note.IndexEntry) bleveDocument {
	return bleveDocument{
		ID:       entry.ID,
		Title:    entry.Title,
		Content:  expandContentForSearch(readEntryContent(entry)),
		Tags:     lowerStrings(entry.Tags),
		Created:  normalizeDate(entry.Created),
		Modified: normalizeDate(entry.Modified),
	}
}

func readEntryContent(entry note.IndexEntry) string {
	if entry.Path == "" {
		return ""
	}
	_, content, err := note.ParseFile(entry.Path)
	if err == nil {
		return content
	}
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return ""
	}
	return string(data)
}

func expandContentForSearch(content string) string {
	var extra []string
	for _, word := range strings.FieldsFunc(content, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	}) {
		lower := strings.ToLower(word)
		if len(lower) > 4 && (strings.HasSuffix(lower, "er") || strings.HasSuffix(lower, "ed") || strings.HasSuffix(lower, "ing")) {
			extra = append(extra, lower[:4])
		}
	}
	if len(extra) == 0 {
		return content
	}
	return content + "\n" + strings.Join(extra, " ")
}

func normalizeDate(s string) string {
	if s == "" {
		return time.Time{}.UTC().Format(time.RFC3339)
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return s
}

func buildBleveQuery(q Query) blevequery.Query {
	var must []blevequery.Query

	if strings.TrimSpace(q.Text) != "" {
		must = append(must, buildTextQuery(q.Text))
	}
	for _, tag := range q.Tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		tq := bleve.NewTermQuery(tag)
		tq.SetField("tags")
		must = append(must, tq)
	}

	if len(must) == 0 {
		return bleve.NewMatchAllQuery()
	}
	if len(must) == 1 {
		return must[0]
	}
	return bleve.NewConjunctionQuery(must...)
}

func buildTextQuery(text string) blevequery.Query {
	text = strings.TrimSpace(text)
	if queryHasExplicitSyntax(text) {
		return bleve.NewQueryStringQuery(withDefaultAND(normalizeQueryFields(text)))
	}
	var termQueries []blevequery.Query
	for _, term := range strings.Fields(text) {
		titleQuery := bleve.NewMatchQuery(term)
		titleQuery.SetField("title")
		titleQuery.SetBoost(4)
		contentQuery := bleve.NewMatchQuery(term)
		contentQuery.SetField("content")
		termQueries = append(termQueries, bleve.NewDisjunctionQuery(titleQuery, contentQuery))
	}
	if len(termQueries) == 1 {
		return termQueries[0]
	}
	return bleve.NewConjunctionQuery(termQueries...)
}

func normalizeQueryFields(text string) string {
	text = strings.ReplaceAll(text, "tag:", "tags:")
	text = strings.ReplaceAll(text, "TAG:", "tags:")
	return text
}

func withDefaultAND(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || queryHasExplicitSyntax(trimmed) {
		return trimmed
	}
	parts := strings.Fields(trimmed)
	for i, part := range parts {
		if strings.HasPrefix(part, "+") || strings.HasPrefix(part, "-") {
			continue
		}
		parts[i] = "+" + part
	}
	return strings.Join(parts, " ")
}

func queryHasExplicitSyntax(text string) bool {
	if strings.ContainsAny(text, "\"():") || strings.Contains(text, "+") || strings.Contains(text, "-") {
		return true
	}
	for _, part := range strings.Fields(text) {
		switch strings.ToUpper(part) {
		case "AND", "OR", "NOT":
			return true
		}
	}
	return false
}

func matchesFromLocations(locations blevesearch.FieldTermLocationMap, q Query) []string {
	set := map[string]bool{}
	if len(q.Tags) > 0 {
		set["tag"] = true
	}
	if q.Text == "" && len(q.Tags) == 0 {
		return []string{"all"}
	}
	for field := range locations {
		if field == "title" || field == "content" {
			set[field] = true
		}
	}
	if q.Text != "" && !set["title"] && !set["content"] {
		set["content"] = true
	}
	matches := make([]string, 0, len(set))
	for match := range set {
		matches = append(matches, match)
	}
	sort.Strings(matches)
	return matches
}

func snippetFromFragments(fragments map[string][]string) string {
	for _, field := range []string{"content", "title"} {
		if values := fragments[field]; len(values) > 0 {
			return strings.Join(values, " … ")
		}
	}
	return ""
}

func lowerStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		lower := strings.ToLower(value)
		out = append(out, lower)
		if _, suffix, ok := strings.Cut(lower, ":"); ok && suffix != "" {
			out = append(out, suffix)
		}
	}
	return out
}
