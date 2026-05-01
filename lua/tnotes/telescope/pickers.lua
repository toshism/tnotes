local M = {}

local pickers = require("telescope.pickers")
local finders = require("telescope.finders")
local conf = require("telescope.config").values
local sorters = require("telescope.sorters")
local actions = require("telescope.actions")
local action_state = require("telescope.actions.state")
local previewers = require("telescope.previewers")

local cli = require("tnotes.cli")
local config = require("tnotes.config")

local DEFAULT_SEARCH_LIMIT = 50
local SNIPPET_LIMIT = 80

-- Get telescope theme options
local function get_theme_opts(opts)
  opts = opts or {}
  local theme = config.options.telescope.theme
  if theme == "dropdown" then
    return require("telescope.themes").get_dropdown(opts)
  elseif theme == "ivy" then
    return require("telescope.themes").get_ivy(opts)
  elseif theme == "cursor" then
    return require("telescope.themes").get_cursor(opts)
  end
  return opts
end

local function display_tags(note)
  local display = {}
  for _, tag in ipairs(note.tags or {}) do
    if not tag:match("^project:") and not tag:match("^path:") then
      table.insert(display, tag)
    end
  end
  if #display == 0 then
    return ""
  end
  return " [" .. table.concat(display, ", ") .. "]"
end

local function clean_snippet(snippet)
  if not snippet or snippet == "" then
    return ""
  end
  snippet = snippet:gsub("<[^>]+>", ""):gsub("%s+", " ")
  snippet = vim.trim(snippet)
  if #snippet > SNIPPET_LIMIT then
    snippet = snippet:sub(1, SNIPPET_LIMIT - 1) .. "…"
  end
  return snippet
end

local function unwrap_result(result)
  local note = result.entry or result
  if result.entry then
    note._tnotes_matches = result.matches
    note._tnotes_snippet = result.snippet
  end
  return note
end

local function normalize_results(results)
  local notes = {}
  for _, result in ipairs(results or {}) do
    table.insert(notes, unwrap_result(result))
  end
  return notes
end

local function sort_recent(notes)
  table.sort(notes, function(a, b)
    return (a.id or "") > (b.id or "")
  end)
  return notes
end

local function note_entry(note)
  local title = note.title or note.id or "untitled"
  local snippet = clean_snippet(note._tnotes_snippet)
  local display = title .. display_tags(note)
  if snippet ~= "" then
    display = display .. " — " .. snippet
  end
  return {
    value = note,
    display = display,
    ordinal = title .. " " .. (note.id or "") .. " " .. table.concat(note.tags or {}, " ") .. " " .. snippet,
    path = note.path,
  }
end

local function note_previewer()
  return previewers.new_buffer_previewer({
    title = "Note Preview",
    define_preview = function(self, entry)
      if entry.path then
        conf.buffer_previewer_maker(entry.path, self.state.bufnr, {
          bufname = self.state.bufname,
        })
      end
    end,
  })
end

local function open_note_mappings(prompt_bufnr, map)
  actions.select_default:replace(function()
    actions.close(prompt_bufnr)
    local selection = action_state.get_selected_entry()
    if selection and selection.path then
      vim.cmd("edit " .. vim.fn.fnameescape(selection.path))
    end
  end)
  return true
end

local function parse_search_prompt(prompt)
  local parsed = { query_parts = {}, tags = {} }
  for _, token in ipairs(vim.split(prompt or "", "%s+", { trimempty = true })) do
    local tag = token:match("^tag:(.+)$")
    local project = token:match("^project:(.+)$")
    if tag and tag ~= "" then
      table.insert(parsed.tags, tag)
    elseif project and project ~= "" then
      parsed.project = project
    else
      table.insert(parsed.query_parts, token)
    end
  end
  parsed.query = table.concat(parsed.query_parts, " ")
  parsed.query_parts = nil
  if #parsed.tags == 0 then
    parsed.tags = nil
  end
  return parsed
end

local function current_project()
  local ok, project = pcall(cli.resolve_project)
  if ok then
    return project
  end
  return nil
end

local function search_notes(prompt)
  local parsed = parse_search_prompt(prompt)
  local opts = {
    query = parsed.query,
    tags = parsed.tags,
    project = parsed.project,
    limit = DEFAULT_SEARCH_LIMIT,
    snippet = true,
  }

  if parsed.query == "" and not parsed.project and not parsed.tags then
    local project = current_project()
    if project and project ~= "" then
      opts.project = project
      local ok, project_results = cli.search(opts)
      if ok and project_results and #project_results > 0 then
        return sort_recent(normalize_results(project_results))
      elseif not ok then
        vim.notify("Failed to search project notes: " .. tostring(project_results), vim.log.levels.WARN)
      end
      opts.project = nil
    end
  end

  local ok, results = cli.search(opts)
  if not ok then
    vim.notify("Failed to search notes: " .. tostring(results), vim.log.levels.ERROR)
    return {}
  end
  local notes = normalize_results(results)
  if parsed.query == "" then
    sort_recent(notes)
  end
  return notes
end

-- Notes picker - indexed search-backed note browser
function M.notes(opts)
  opts = get_theme_opts(opts)

  pickers.new(opts, {
    prompt_title = "tnotes search",
    finder = finders.new_dynamic({
      fn = search_notes,
      entry_maker = note_entry,
    }),
    sorter = sorters.empty(),
    previewer = note_previewer(),
    attach_mappings = open_note_mappings,
  }):find()
end

-- Legacy all-notes picker - flat list fuzzy find by title
function M.notes_all(opts)
  opts = get_theme_opts(opts)

  local ok, notes = cli.list()
  if not ok then
    vim.notify("Failed to list notes: " .. tostring(notes), vim.log.levels.ERROR)
    return
  end

  pickers.new(opts, {
    prompt_title = "tnotes all notes",
    finder = finders.new_table({
      results = notes,
      entry_maker = note_entry,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = note_previewer(),
    attach_mappings = open_note_mappings,
  }):find()
end

-- Project notes picker - notes filtered by auto-detected project
function M.project_notes(opts)
  opts = get_theme_opts(opts)

  local project = cli.resolve_project()
  local ok, results = cli.list(project)
  if not ok then
    vim.notify("Failed to list project notes: " .. tostring(results), vim.log.levels.ERROR)
    return
  end

  local notes = normalize_results(results)

  pickers.new(opts, {
    prompt_title = "tnotes [" .. project .. "]",
    finder = finders.new_table({
      results = notes,
      entry_maker = note_entry,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = note_previewer(),
    attach_mappings = open_note_mappings,
  }):find()
end

-- Tags picker - browse all tags
function M.tags(opts)
  opts = get_theme_opts(opts)

  local ok, tags = cli.tags()
  if not ok then
    vim.notify("Failed to get tags: " .. tostring(tags), vim.log.levels.ERROR)
    return
  end

  pickers.new(opts, {
    prompt_title = "tnotes tags",
    finder = finders.new_table({
      results = tags,
      entry_maker = function(item)
        return {
          value = item,
          display = item.tag .. " (" .. item.count .. ")",
          ordinal = item.tag,
        }
      end,
    }),
    sorter = conf.generic_sorter(opts),
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection then
          -- Open notes filtered by this tag
          M.notes_by_tag(selection.value.tag)
        end
      end)
      return true
    end,
  }):find()
end

-- Notes filtered by tag
function M.notes_by_tag(tag, opts)
  opts = get_theme_opts(opts)

  local ok, notes = cli.search(nil, tag)
  if not ok then
    vim.notify("Failed to search notes: " .. tostring(notes), vim.log.levels.ERROR)
    return
  end

  pickers.new(opts, {
    prompt_title = "tnotes [tag: " .. tag .. "]",
    finder = finders.new_table({
      results = normalize_results(notes),
      entry_maker = note_entry,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = note_previewer(),
    attach_mappings = open_note_mappings,
  }):find()
end

-- Search picker - full text search
function M.search(opts)
  opts = get_theme_opts(opts)

  -- Get initial query from user
  local query = vim.fn.input("Search notes: ")
  if query == "" then
    return
  end

  local ok, notes = cli.search({ query = query, snippet = true })
  if not ok then
    vim.notify("Failed to search notes: " .. tostring(notes), vim.log.levels.ERROR)
    return
  end

  pickers.new(opts, {
    prompt_title = "tnotes search: " .. query,
    finder = finders.new_table({
      results = normalize_results(notes),
      entry_maker = note_entry,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = note_previewer(),
    attach_mappings = open_note_mappings,
  }):find()
end

return M
