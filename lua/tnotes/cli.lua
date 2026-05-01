local M = {}

local config = require("tnotes.config")

-- Build command with appropriate flags
local function build_cmd(args)
  local cmd = { config.options.bin }

  for _, arg in ipairs(args) do
    table.insert(cmd, arg)
  end

  return cmd
end

-- Run tnotes command and return parsed JSON
-- Returns: success (bool), result (table or error string)
function M.run(args, opts)
  opts = opts or {}
  local cmd = build_cmd(args)

  -- Add --json flag if not already present
  local has_json = false
  for _, arg in ipairs(args) do
    if arg == "--json" then
      has_json = true
      break
    end
  end
  if not has_json and opts.json ~= false then
    table.insert(cmd, "--json")
  end

  local result = vim.fn.system(cmd)
  local exit_code = vim.v.shell_error

  if exit_code ~= 0 then
    return false, result
  end

  -- Parse JSON if expected
  if opts.json ~= false then
    local ok, parsed = pcall(vim.fn.json_decode, result)
    if not ok then
      return false, "Failed to parse JSON: " .. result
    end
    return true, parsed
  end

  return true, result
end

-- Resolve project name for current working directory
-- Checks: .tnotes-project (walking up) > git remote origin > cwd basename
function M.resolve_project()
  local cwd = vim.fn.getcwd()

  -- Walk up looking for .tnotes-project
  local dir = cwd
  while true do
    local dotfile = dir .. "/.tnotes-project"
    local f = io.open(dotfile, "r")
    if f then
      local name = f:read("*l")
      f:close()
      if name and name ~= "" then
        return vim.trim(name)
      end
    end
    local parent = vim.fn.fnamemodify(dir, ":h")
    if parent == dir then
      break
    end
    dir = parent
  end

  -- Try git remote origin
  local remote = vim.fn.system("git remote get-url origin 2>/dev/null")
  if vim.v.shell_error == 0 and remote ~= "" then
    remote = vim.trim(remote)
    -- Normalize: git@host:org/repo.git or https://host/org/repo.git -> org/repo
    remote = remote:gsub("%.git$", "")
    local org_repo = remote:match("^git@[^:]+:(.+)$")
    if not org_repo then
      org_repo = remote:match("^https?://[^/]+/(.+)$")
    end
    if org_repo then
      return org_repo
    end
  end

  -- Fallback: cwd basename
  return vim.fn.fnamemodify(cwd, ":t")
end

-- List all notes, optionally filtered by project
function M.list(project)
  if project and project ~= "" then
    return M.search({ project = project, limit = 0 })
  end
  return M.run({ "list" })
end

-- Search notes
-- Accepts either an opts table:
--   { query = "text", tags = { "tag" }, project = "name", limit = 20, snippet = true }
-- or the legacy positional shape: search(query, tag).
function M.search(opts, legacy_tag)
  if type(opts) == "string" or legacy_tag ~= nil then
    opts = { query = opts, tags = legacy_tag }
  elseif opts == nil then
    opts = {}
  end

  local args = { "search" }
  if opts.query and opts.query ~= "" then
    table.insert(args, opts.query)
  end

  local tags = opts.tags or opts.tag
  if type(tags) == "table" then
    tags = table.concat(tags, ",")
  end
  if tags and tags ~= "" then
    table.insert(args, "--tag")
    table.insert(args, tags)
  end

  if opts.project and opts.project ~= "" then
    table.insert(args, "--project")
    table.insert(args, opts.project)
  end

  if opts.limit ~= nil then
    table.insert(args, "--limit")
    table.insert(args, tostring(opts.limit))
  end

  if opts.snippet then
    table.insert(args, "--snippet")
  end

  return M.run(args)
end

-- Show a note by ID
function M.show(id)
  return M.run({ "show", id }, { json = false })
end

-- Add a new note
function M.add(title, tags)
  local args = { "add", title }
  if tags and tags ~= "" then
    table.insert(args, "--tags")
    table.insert(args, tags)
  end
  return M.run(args)
end

-- Get all unique tags
function M.tags()
  local ok, notes = M.list()
  if not ok then
    return false, notes
  end

  local tag_counts = {}
  for _, note in ipairs(notes) do
    if note.tags then
      for _, tag in ipairs(note.tags) do
        tag_counts[tag] = (tag_counts[tag] or 0) + 1
      end
    end
  end

  -- Convert to list
  local tags = {}
  for tag, count in pairs(tag_counts) do
    table.insert(tags, { tag = tag, count = count })
  end

  -- Sort by count descending
  table.sort(tags, function(a, b)
    return a.count > b.count
  end)

  return true, tags
end

-- Rebuild the index
function M.index()
  return M.run({ "index" }, { json = false })
end

return M
