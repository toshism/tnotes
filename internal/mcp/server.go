package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/toshism/tnotes/internal/config"
	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/note"
	"github.com/toshism/tnotes/internal/project"
	"github.com/toshism/tnotes/internal/search"
)

// JSON-RPC types
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP types
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct{}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server handles MCP protocol
type Server struct {
	in  io.Reader
	out io.Writer
}

// NewServer creates a new MCP server
func NewServer() *Server {
	return &Server{
		in:  os.Stdin,
		out: os.Stdout,
	}
}

// Run starts the MCP server loop
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.in)
	// Increase buffer size for large messages
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// No response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *Request) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "tnotes",
			Version: "0.1.0",
		},
		Capabilities: Capabilities{
			Tools: &ToolsCapability{},
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req *Request) {
	workflowInfo := `

WORKFLOW:
- All notes are stored in a single directory (default ~/tnotes, configurable)
- Notes are automatically tagged with project:<name> and path:<cwd> on creation
- Search across all notes by default, or filter by project
- To EDIT notes: use the returned 'path' with Read/Edit tools directly`

	tools := []Tool{
		{
			Name:        "tnotes_init",
			Description: "Initialize the tnotes directory. Creates the .tnotes folder with an empty index." + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "tnotes_list",
			Description: "List all notes. Returns JSON array with id, title, tags, and absolute file path for each note. Use 'project' to filter by project tag.\n\nWARNING: Without a project filter this can return a very large result. Prefer using 'project' to filter, or use tnotes_search to find specific notes." + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project": {Type: "string", Description: "Filter notes by project name"},
				},
			},
		},
		{
			Name:        "tnotes_search",
			Description: "Search notes by text (title + content) and/or tags. Returns matching notes with id, title, tags, and absolute file path." + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":   {Type: "string", Description: "Text to search for in title and content"},
					"tag":     {Type: "string", Description: "Filter by tag (exact match)"},
					"project": {Type: "string", Description: "Filter by project name"},
				},
			},
		},
		{
			Name:        "tnotes_show",
			Description: "Get the full content of a note by ID. Returns note metadata and full content. Use the 'path' field with Read/Edit tools to modify." + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {Type: "string", Description: "The note ID"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name: "tnotes_add",
			Description: `Create a new note with the given title and optional tags. Notes are automatically tagged with project:<name> and path:<cwd>. Returns the new note's ID, absolute path, and resolved project info.` + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title":   {Type: "string", Description: "The note title"},
					"tags":    {Type: "string", Description: "Comma-separated list of tags"},
					"content": {Type: "string", Description: "Initial content for the note body"},
					"links":   {Type: "string", Description: "Comma-separated filenames of related notes (e.g. '20260218143022-my-note.md, 20260217120000-other-note.md')"},
					"project": {Type: "string", Description: "Project name to associate with this note. If empty, auto-detected from .tnotes-project file, git remote, or directory name."},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "tnotes_index",
			Description: "Rebuild the index from all markdown files in the notes directory. Run this after manually editing/adding/deleting note files to update the search index." + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
			},
		},
	}

	s.sendResult(req.ID, ToolsListResult{Tools: tools})
}

func (s *Server) handleToolsCall(req *Request) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	notesDir := config.NotesDir
	var result string
	var isError bool

	switch params.Name {
	case "tnotes_init":
		result, isError = s.toolInit(notesDir)
	case "tnotes_list":
		projectFilter, _ := params.Arguments["project"].(string)
		result, isError = s.toolList(notesDir, projectFilter)
	case "tnotes_search":
		query, _ := params.Arguments["query"].(string)
		tag, _ := params.Arguments["tag"].(string)
		projectFilter, _ := params.Arguments["project"].(string)
		result, isError = s.toolSearch(notesDir, query, tag, projectFilter)
	case "tnotes_show":
		id, _ := params.Arguments["id"].(string)
		result, isError = s.toolShow(notesDir, id)
	case "tnotes_add":
		title, _ := params.Arguments["title"].(string)
		tags, _ := params.Arguments["tags"].(string)
		content, _ := params.Arguments["content"].(string)
		links, _ := params.Arguments["links"].(string)
		projectParam, _ := params.Arguments["project"].(string)
		result, isError = s.toolAdd(notesDir, title, tags, content, links, projectParam)
	case "tnotes_index":
		result, isError = s.toolIndex(notesDir)
	default:
		result = fmt.Sprintf("Unknown tool: %s", params.Name)
		isError = true
	}

	s.sendResult(req.ID, ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
		IsError: isError,
	})
}

func (s *Server) toolInit(notesDir string) (string, bool) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return fmt.Sprintf("Failed to resolve path: %v", err), true
	}

	tnotesDir := filepath.Join(absDir, ".tnotes")
	indexFile := filepath.Join(tnotesDir, "index.json")

	if _, err := os.Stat(indexFile); err == nil {
		out := map[string]interface{}{
			"status":  "already_initialized",
			"path":    absDir,
			"message": "Notes directory already initialized",
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		return string(data), false
	}

	if err := os.MkdirAll(tnotesDir, 0755); err != nil {
		return fmt.Sprintf("Failed to create directory: %v", err), true
	}

	emptyIndex := &index.Index{Entries: []note.IndexEntry{}}
	if err := saveIndexTo(emptyIndex, indexFile); err != nil {
		return fmt.Sprintf("Failed to create index: %v", err), true
	}

	out := map[string]interface{}{
		"status":  "initialized",
		"path":    absDir,
		"message": fmt.Sprintf("Initialized tnotes at %s", absDir),
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data), false
}

func (s *Server) toolList(notesDir, projectFilter string) (string, bool) {
	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	idx, err := loadIndexFrom(indexFile)
	if err != nil {
		return fmt.Sprintf("Failed to load index: %v", err), true
	}

	entries := idx.Entries
	if projectFilter != "" {
		var filtered []note.IndexEntry
		projectTag := "project:" + projectFilter
		for _, e := range entries {
			for _, t := range e.Tags {
				if t == projectTag {
					filtered = append(filtered, e)
					break
				}
			}
		}
		entries = filtered
	}

	data, _ := json.MarshalIndent(entries, "", "  ")
	return string(data), false
}

func (s *Server) toolSearch(notesDir, query, tag, projectFilter string) (string, bool) {
	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	idx, err := loadIndexFrom(indexFile)
	if err != nil {
		return fmt.Sprintf("Failed to load index: %v", err), true
	}

	q := search.Query{Text: query}
	if tag != "" {
		q.Tags = []string{tag}
	}
	if projectFilter != "" {
		q.Tags = append(q.Tags, "project:"+projectFilter)
	}

	results, err := search.Search(idx, q)
	if err != nil {
		return fmt.Sprintf("Search failed: %v", err), true
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return string(data), false
}

func (s *Server) toolShow(notesDir, id string) (string, bool) {
	if id == "" {
		return "ID is required", true
	}

	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	idx, _ := loadIndexFrom(indexFile)
	var filePath string

	if idx != nil {
		if entry := idx.FindByID(id); entry != nil {
			filePath = entry.Path
		}
	}

	var n *note.Note
	var content string
	var err error

	if filePath != "" {
		n, content, err = note.ParseFile(filePath)
	} else {
		n, content, err = note.FindNoteByID(notesDir, id)
	}

	if err != nil {
		return fmt.Sprintf("Note not found: %v", err), true
	}

	out := map[string]interface{}{
		"id":       n.ID,
		"title":    n.Title,
		"tags":     n.Tags,
		"links":    n.Links,
		"created":  n.Created,
		"modified": n.Modified,
		"path":     n.Path,
		"content":  content,
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data), false
}

func (s *Server) toolAdd(notesDir, title, tags, content, links, projectParam string) (string, bool) {
	if title == "" {
		return "Title is required", true
	}

	// Resolve project and path
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("Failed to get working directory: %v", err), true
	}

	projectName := projectParam
	if projectName == "" {
		info := project.Resolve(cwd)
		projectName = info.Project
	}

	var tagList []string
	if tags != "" {
		for _, t := range splitAndTrim(tags, ",") {
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	// Add project and path tags
	tagList = append(tagList, "project:"+projectName, "path:"+cwd)

	n := note.NewNote(title, tagList)

	if links != "" {
		for _, l := range splitAndTrim(links, ",") {
			if l != "" {
				n.Links = append(n.Links, l)
			}
		}
	}

	filename := n.Filename()
	absDir, _ := filepath.Abs(notesDir)
	filePath := filepath.Join(absDir, filename)

	noteContent := n.ToMarkdown(content)
	if err := os.WriteFile(filePath, []byte(noteContent), 0644); err != nil {
		return fmt.Sprintf("Failed to write note: %v", err), true
	}

	n.Path = filePath
	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	idx, err := loadIndexFrom(indexFile)
	if err != nil {
		idx = &index.Index{Entries: []note.IndexEntry{}}
	}

	idx.AddEntry(n.ToIndexEntry())
	if err := saveIndexTo(idx, indexFile); err != nil {
		return fmt.Sprintf("Failed to save index: %v", err), true
	}

	out := map[string]interface{}{
		"id":      n.ID,
		"title":   n.Title,
		"path":    filePath,
		"tags":    n.Tags,
		"project": projectName,
		"cwd":     cwd,
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data), false
}

func (s *Server) toolIndex(notesDir string) (string, bool) {
	idx, err := rebuildIndexFor(notesDir)
	if err != nil {
		return fmt.Sprintf("Failed to rebuild index: %v", err), true
	}

	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	if err := saveIndexTo(idx, indexFile); err != nil {
		return fmt.Sprintf("Failed to save index: %v", err), true
	}

	out := map[string]interface{}{
		"status": "rebuilt",
		"count":  len(idx.Entries),
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data), false
}

// loadIndexFrom loads an index from a specific file path
func loadIndexFrom(indexFile string) (*index.Index, error) {
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	var idx index.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	return &idx, nil
}

// saveIndexTo saves an index to a specific file path
func saveIndexTo(idx *index.Index, indexFile string) error {
	indexDir := filepath.Dir(indexFile)
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, data, 0644)
}

// rebuildIndexFor rebuilds the index for a specific notes directory
func rebuildIndexFor(notesDir string) (*index.Index, error) {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return nil, err
	}

	idx := &index.Index{Entries: []note.IndexEntry{}}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() && info.Name() == ".tnotes" {
			return filepath.SkipDir
		}

		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		n, _, err := note.ParseFile(path)
		if err != nil {
			return nil
		}

		idx.Entries = append(idx.Entries, n.ToIndexEntry())
		return nil
	})

	if err != nil {
		return nil, err
	}

	return idx, nil
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
	s.send(resp)
}

func (s *Server) send(resp Response) {
	data, _ := json.Marshal(resp)
	fmt.Fprintln(s.out, string(data))
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
