package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tosh/tnotes/internal/config"
	"github.com/tosh/tnotes/internal/index"
	"github.com/tosh/tnotes/internal/note"
	"github.com/tosh/tnotes/internal/search"
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

// resolveNotesDir returns the appropriate notes directory based on path/global flags
// Priority: path > global > auto-detect (config.NotesDir)
func resolveNotesDir(path string, global bool) string {
	if path != "" {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return config.NotesDir
		}
		return filepath.Join(home, "notes")
	}
	return config.NotesDir
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
	// Common properties for directory selection
	dirProps := map[string]Property{
		"global": {Type: "boolean", Description: "If true, use global ~/notes instead of project notes"},
		"dir":    {Type: "string", Description: "Explicit path to notes directory (overrides global)"},
	}

	workflowInfo := `

WORKFLOW:
- tnotes supports TWO note locations: project-local (./tnotes/) and global (~/notes)
- Auto-detection: if ./tnotes/.tnotes exists in cwd, uses project notes; otherwise falls back to global
- For PROJECT notes: first check if initialized. If not, use tnotes_init to create ./tnotes/.tnotes
- For GLOBAL notes: use 'global: true' parameter
- To EDIT notes: use the returned 'path' with Read/Edit tools directly

IMPORTANT - When editing notes:
- Update the 'modified' field in frontmatter to today's date (YYYY-MM-DD format)
- When adding wikilinks like [[note-id]] in content, ALSO add the note-id to the 'links' array in frontmatter`

	tools := []Tool{
		{
			Name: "tnotes_init",
			Description: `Initialize a new tnotes directory. Creates the .tnotes folder with an empty index.

USE THIS FIRST when you want to create project-specific notes but ./tnotes/.tnotes doesn't exist yet.

Without arguments: initializes ./tnotes/ in current working directory (for project notes).
With 'dir': initializes the specified directory.
With 'global: true': initializes ~/notes (usually already exists).` + workflowInfo,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: dirProps,
			},
		},
		{
			Name: "tnotes_list",
			Description: `List all notes in the notes directory.

Returns JSON array with id, title, tags, and absolute file path for each note.

If no project notes exist (./tnotes/.tnotes), falls back to global ~/notes.
Use 'global: true' to explicitly list global notes.` + workflowInfo,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: dirProps,
			},
		},
		{
			Name: "tnotes_search",
			Description: `Search notes by text (title + content) and/or tags.

Returns matching notes with id, title, tags, and absolute file path.` + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":  {Type: "string", Description: "Text to search for in title and content"},
					"tag":    {Type: "string", Description: "Filter by tag (exact match)"},
					"global": {Type: "boolean", Description: "If true, use global ~/notes instead of project notes"},
					"dir":    {Type: "string", Description: "Explicit path to notes directory (overrides global)"},
				},
			},
		},
		{
			Name: "tnotes_show",
			Description: `Get the full content of a note by ID.

Returns note metadata and full content. Use the 'path' field with Read/Edit tools to modify.` + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id":     {Type: "string", Description: "The note ID"},
					"global": {Type: "boolean", Description: "If true, search in global ~/notes"},
					"dir":    {Type: "string", Description: "Explicit path to notes directory"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name: "tnotes_add",
			Description: `Create a new note with the given title and optional tags.

Returns the new note's ID and absolute path.

USE 'project: true' when user wants a project-level note. This will:
1. Auto-initialize ./tnotes/.tnotes if it doesn't exist
2. Create the note in project notes (./tnotes/)

USE 'global: true' when user wants a global note (~/notes).

If neither is specified, auto-detects (project if exists, else global).` + workflowInfo,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title":   {Type: "string", Description: "The note title"},
					"tags":    {Type: "string", Description: "Comma-separated list of tags"},
					"content": {Type: "string", Description: "Initial content for the note body"},
					"project": {Type: "boolean", Description: "If true, create as PROJECT note (auto-initializes ./tnotes/ if needed)"},
					"global":  {Type: "boolean", Description: "If true, create in global ~/notes"},
					"dir":     {Type: "string", Description: "Explicit path to notes directory"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name: "tnotes_index",
			Description: `Rebuild the index from all markdown files in the notes directory.

Run this after manually editing/adding/deleting note files to update the search index.` + workflowInfo,
			InputSchema: InputSchema{
				Type:       "object",
				Properties: dirProps,
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

	// Extract common dir params
	global, _ := params.Arguments["global"].(bool)
	dir, _ := params.Arguments["dir"].(string)
	notesDir := resolveNotesDir(dir, global)

	var result string
	var isError bool

	switch params.Name {
	case "tnotes_init":
		result, isError = s.toolInit(notesDir, global, dir)
	case "tnotes_list":
		result, isError = s.toolList(notesDir)
	case "tnotes_search":
		query, _ := params.Arguments["query"].(string)
		tag, _ := params.Arguments["tag"].(string)
		result, isError = s.toolSearch(notesDir, query, tag)
	case "tnotes_show":
		id, _ := params.Arguments["id"].(string)
		result, isError = s.toolShow(notesDir, id)
	case "tnotes_add":
		title, _ := params.Arguments["title"].(string)
		tags, _ := params.Arguments["tags"].(string)
		content, _ := params.Arguments["content"].(string)
		project, _ := params.Arguments["project"].(bool)
		result, isError = s.toolAdd(notesDir, title, tags, content, project, global, dir)
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

func (s *Server) toolInit(notesDir string, global bool, explicitDir string) (string, bool) {
	// Determine the target directory
	var targetDir string
	if explicitDir != "" {
		targetDir = explicitDir
	} else if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Sprintf("Failed to get home directory: %v", err), true
		}
		targetDir = filepath.Join(home, "notes")
	} else {
		// Default: create ./tnotes in current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Sprintf("Failed to get working directory: %v", err), true
		}
		targetDir = filepath.Join(cwd, "tnotes")
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Sprintf("Failed to resolve path: %v", err), true
	}

	tnotesDir := filepath.Join(absDir, ".tnotes")
	indexFile := filepath.Join(tnotesDir, "index.json")

	// Check if already initialized
	if _, err := os.Stat(indexFile); err == nil {
		out := map[string]interface{}{
			"status":  "already_initialized",
			"path":    absDir,
			"message": "Project notes already initialized at this location",
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		return string(data), false
	}

	// Create directory structure
	if err := os.MkdirAll(tnotesDir, 0755); err != nil {
		return fmt.Sprintf("Failed to create directory: %v", err), true
	}

	// Create empty index
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

func (s *Server) toolList(notesDir string) (string, bool) {
	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	idx, err := loadIndexFrom(indexFile)
	if err != nil {
		return fmt.Sprintf("Failed to load index: %v", err), true
	}

	data, _ := json.MarshalIndent(idx.Entries, "", "  ")
	return string(data), false
}

func (s *Server) toolSearch(notesDir, query, tag string) (string, bool) {
	indexFile := filepath.Join(notesDir, ".tnotes", "index.json")
	idx, err := loadIndexFrom(indexFile)
	if err != nil {
		return fmt.Sprintf("Failed to load index: %v", err), true
	}

	q := search.Query{Text: query}
	if tag != "" {
		q.Tags = []string{tag}
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

func (s *Server) toolAdd(notesDir, title, tags, content string, project, global bool, explicitDir string) (string, bool) {
	if title == "" {
		return "Title is required", true
	}

	// Handle project flag - auto-initialize if needed
	targetDir := notesDir
	if project && explicitDir == "" && !global {
		// User explicitly wants a project note
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Sprintf("Failed to get working directory: %v", err), true
		}
		projectDir := filepath.Join(cwd, "tnotes")
		tnotesIndex := filepath.Join(projectDir, ".tnotes", "index.json")

		// Auto-initialize if not exists
		if _, err := os.Stat(tnotesIndex); os.IsNotExist(err) {
			tnotesDir := filepath.Join(projectDir, ".tnotes")
			if err := os.MkdirAll(tnotesDir, 0755); err != nil {
				return fmt.Sprintf("Failed to create project notes directory: %v", err), true
			}
			emptyIndex := &index.Index{Entries: []note.IndexEntry{}}
			if err := saveIndexTo(emptyIndex, tnotesIndex); err != nil {
				return fmt.Sprintf("Failed to initialize project notes: %v", err), true
			}
		}
		targetDir = projectDir
	}

	var tagList []string
	if tags != "" {
		for _, t := range splitAndTrim(tags, ",") {
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	n := note.NewNote(title, tagList)
	filename := n.Filename()
	absDir, _ := filepath.Abs(targetDir)
	filePath := filepath.Join(absDir, filename)

	noteContent := n.ToMarkdown(content)
	if err := os.WriteFile(filePath, []byte(noteContent), 0644); err != nil {
		return fmt.Sprintf("Failed to write note: %v", err), true
	}

	n.Path = filePath
	indexFile := filepath.Join(targetDir, ".tnotes", "index.json")
	idx, err := loadIndexFrom(indexFile)
	if err != nil {
		// Create new index if it doesn't exist
		idx = &index.Index{Entries: []note.IndexEntry{}}
	}

	idx.AddEntry(n.ToIndexEntry())
	if err := saveIndexTo(idx, indexFile); err != nil {
		return fmt.Sprintf("Failed to save index: %v", err), true
	}

	location := "auto-detected"
	if project {
		location = "project"
	} else if global {
		location = "global"
	}

	out := map[string]interface{}{
		"id":       n.ID,
		"title":    n.Title,
		"path":     filePath,
		"tags":     n.Tags,
		"location": location,
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
