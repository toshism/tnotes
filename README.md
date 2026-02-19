# tnotes

AI-native note-taking. CLI, MCP server, and Neovim plugin.

- Plain markdown files with YAML frontmatter
- JSON index for fast search
- Automatic project tagging (detects from `.tnotes-project`, git remote, or directory name)
- MCP server for AI assistants (Claude Code, Claude Desktop, etc.)
- Neovim plugin with Telescope integration

## Install

```
go install github.com/toshism/tnotes@latest
```

## Quick start

```bash
# Initialize the notes directory (default: ~/tnotes)
tnotes init

# Create a note
tnotes add "My first note"

# List all notes
tnotes list

# Search notes
tnotes search "query"

# Show a note by ID
tnotes show 20260219143022

# Rebuild the index after manual edits
tnotes index
```

All commands support `--json` for machine-readable output.

Notes are stored as markdown files in `~/tnotes` (configurable via `--dir`, `$TNOTES_DIR`, or `~/.config/tnotes/config.toml`).

## MCP server

tnotes includes an MCP server so AI assistants can read, search, and create notes directly.

### Claude Code

Add to your MCP settings (`~/.claude/claude_desktop_config.json` or project `.claude/settings.json`):

```json
{
  "mcpServers": {
    "tnotes": {
      "command": "tnotes",
      "args": ["mcp"]
    }
  }
}
```

To use a custom notes directory:

```json
{
  "mcpServers": {
    "tnotes": {
      "command": "tnotes",
      "args": ["mcp", "--dir", "/path/to/notes"]
    }
  }
}
```

### Available MCP tools

| Tool | Description |
|------|-------------|
| `tnotes_init` | Initialize the notes directory |
| `tnotes_list` | List notes, optionally filtered by project |
| `tnotes_search` | Search by text and/or tags |
| `tnotes_show` | Get full note content by ID |
| `tnotes_add` | Create a new note |
| `tnotes_index` | Rebuild the search index |

## Neovim plugin

Requires [telescope.nvim](https://github.com/nvim-telescope/telescope.nvim) and the `tnotes` binary in your `$PATH`.

### Install with lazy.nvim

```lua
{
  "toshism/tnotes",
  dependencies = {
    "nvim-telescope/telescope.nvim",
    "nvim-lua/plenary.nvim",
  },
  config = function()
    require("tnotes").setup()
  end,
}
```

### Default keymaps

| Key | Action |
|-----|--------|
| `gf` | Follow tnotes link (in markdown files) |
| `<leader>nn` | Create new note |
| `<leader>ns` | Search notes (Telescope) |
| `<leader>nt` | Browse tags (Telescope) |

All keymaps can be customized or disabled:

```lua
require("tnotes").setup({
  mappings = {
    follow_link = "gf",
    new_note = "<leader>nn",
    search = "<leader>ns",
    tags = "<leader>nt",
    -- set to false to disable any mapping
  },
})
```

## Project tagging

When creating notes, tnotes automatically tags them with the current project. It resolves the project name in this order:

1. `.tnotes-project` file (walks up to filesystem root)
2. Git remote origin (normalized to `org/repo`)
3. Current directory basename

## CLAUDE.md integration

Add tnotes to your `CLAUDE.md` so Claude uses it for working notes:

```markdown
## Document Creation

- **Prefer tnotes** for drafts, research documents, review documents, and working notes
  unless the output has a specific designated location.
- Use the `project` parameter to tag notes with the relevant project name.
- Omit `project` to let tnotes auto-detect from `.tnotes-project`, git remote, or directory name.
```

## CLI reference

| Command | Description |
|---------|-------------|
| `tnotes init` | Initialize notes directory and index |
| `tnotes add <title>` | Create a new note (`-t` tags, `-p` project) |
| `tnotes list` | List all notes |
| `tnotes show <id>` | Display a note by ID |
| `tnotes search <query>` | Search notes (`--tag` to filter by tag) |
| `tnotes index` | Rebuild the index from files on disk |
| `tnotes mcp` | Run as MCP server (stdio) |

## License

MIT
