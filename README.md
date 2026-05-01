# tnotes

AI-native note-taking for plain Markdown files.

`tnotes` is a small Go CLI with an MCP server and a Neovim/Telescope plugin.
Notes live as regular `.md` files with YAML frontmatter, while `.tnotes/` stores the derived search indexes.

## Features

- plain Markdown notes with YAML frontmatter
- timestamp-based note IDs
- fast indexed search over titles and content
- automatic `project:<name>` and `path:<cwd>` tags on note creation
- MCP server for AI assistants
- Neovim plugin with Telescope pickers

## Install

### CLI

```bash
go install github.com/toshism/tnotes@latest
```

### Neovim

Requires:

- `tnotes` in your `$PATH`
- [`telescope.nvim`](https://github.com/nvim-telescope/telescope.nvim)
- [`plenary.nvim`](https://github.com/nvim-lua/plenary.nvim)

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

## Quick start

```bash
# initialize the notes directory (default: ~/tnotes)
tnotes init

# create a note
tnotes add "My first note"

# create a note with tags
tnotes add "Search ideas" --tags search,bleve

# list notes
tnotes list

# search notes
tnotes search "query"
tnotes search --tag meeting
tnotes search --project toshism/tnotes "index rebuild"

# show a note by ID
tnotes show 20260219143022

# rebuild indexes after manual edits
tnotes index
```

All commands support `--json`.

## Storage and config

By default notes are stored in `~/tnotes`.

You can override the notes directory with, in priority order:

1. `--dir`
2. `TNOTES_DIR`
3. `~/.config/tnotes/config.toml`

Example config:

```toml
notes_dir = "~/notes/tnotes"
```

Each notes directory contains:

- your `*.md` note files
- `.tnotes/index.json` metadata index
- `.tnotes/bleve/` full-text search index

## Project tagging

When creating notes, `tnotes` resolves the project name in this order:

1. `.tnotes-project` file found by walking upward
2. git remote `origin`, normalized to `org/repo`
3. current directory basename

New notes automatically get:

- `project:<resolved-project>`
- `path:<current-working-directory>`

You can also set the project explicitly with `tnotes add --project <name>`.

## Search

`tnotes search` searches indexed note titles and content.

Examples:

```bash
tnotes search "roadmap"
tnotes search --tag meeting
tnotes search --project toshism/tnotes "bleve"
tnotes search --snippet "index"
```

The Neovim Telescope picker also supports prompt prefixes like:

- `tag:meeting`
- `project:toshism/tnotes`
- `tag:meeting project:toshism/tnotes roadmap`

## MCP server

Run the stdio MCP server with:

```bash
tnotes mcp
```

Claude Code / Claude Desktop config:

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

Custom notes directory:

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

Available MCP tools:

- `tnotes_init`
- `tnotes_list`
- `tnotes_search`
- `tnotes_show`
- `tnotes_add`
- `tnotes_index`

## Neovim

Default mappings:

| Key | Action |
|-----|--------|
| `gf` | Follow `[[wiki links]]` in Markdown, otherwise normal `gf` |
| `<leader>nn` | Create a new note |
| `<leader>ns` | Open the main indexed notes picker |
| `<leader>nt` | Browse tags |

User commands:

- `:TnotesNew [title]`
- `:TnotesRefresh`
- `:Telescope tnotes notes`
- `:Telescope tnotes notes_all`
- `:Telescope tnotes tags`
- `:Telescope tnotes search`

Example config:

```lua
require("tnotes").setup({
  bin = "tnotes",
  mappings = {
    follow_link = "gf",
    new_note = "<leader>nn",
    search = "<leader>ns",
    tags = "<leader>nt",
  },
  telescope = {
    theme = nil, -- or "dropdown", "ivy", "cursor"
  },
})
```

## CLI reference

| Command | Description |
|---------|-------------|
| `tnotes init` | Initialize a notes directory |
| `tnotes add <title>` | Create a note |
| `tnotes list` | List notes |
| `tnotes show <id>` | Print a note |
| `tnotes search [query]` | Search notes |
| `tnotes index` | Rebuild metadata and search indexes |
| `tnotes mcp` | Run the MCP server over stdio |

## License

MIT
