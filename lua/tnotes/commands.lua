local M = {}

local cli = require("tnotes.cli")

-- Create a new note and open it
function M.new(opts)
  opts = opts or {}
  local title = opts.args

  if not title or title == "" then
    title = vim.fn.input("Note title: ")
    if title == "" then
      return
    end
  end

  local ok, result = cli.add(title, opts.tags)
  if not ok then
    vim.notify("Failed to create note: " .. tostring(result), vim.log.levels.ERROR)
    return
  end

  -- Open the new note
  if result and result.path then
    vim.cmd("edit " .. vim.fn.fnameescape(result.path))
  end
end

-- Follow link under cursor
function M.follow_link()
  local line = vim.api.nvim_get_current_line()
  local col = vim.api.nvim_win_get_cursor(0)[2]

  -- Find [[...]] pattern around cursor
  local pattern = "%[%[([^%]]+)%]%]"
  local start_pos = 1

  while true do
    local match_start, match_end, link_content = line:find(pattern, start_pos)
    if not match_start then
      break
    end

    -- Check if cursor is within this match
    if col >= match_start - 1 and col <= match_end - 1 then
      -- Found link under cursor
      return M.open_link(link_content)
    end

    start_pos = match_end + 1
  end

  -- No link found, fall back to default gf behavior
  vim.cmd("normal! gf")
end

-- Open a note by ID or title
function M.open_link(link)
  -- First try as ID (numeric)
  local ok, notes = cli.list()
  if not ok then
    vim.notify("Failed to list notes: " .. tostring(notes), vim.log.levels.ERROR)
    return
  end

  for _, note in ipairs(notes) do
    -- Match by ID
    if note.id == link then
      vim.cmd("edit " .. vim.fn.fnameescape(note.path))
      return
    end
    -- Match by title (case-insensitive)
    if note.title and note.title:lower() == link:lower() then
      vim.cmd("edit " .. vim.fn.fnameescape(note.path))
      return
    end
  end

  vim.notify("Note not found: " .. link, vim.log.levels.WARN)
end

-- Refresh/rebuild the index
function M.refresh()
  local ok, result = cli.index()
  if ok then
    vim.notify("tnotes: index refreshed", vim.log.levels.INFO)
  else
    vim.notify("tnotes: refresh failed - " .. tostring(result), vim.log.levels.ERROR)
  end
end

-- Register all commands
function M.setup()
  vim.api.nvim_create_user_command("TnotesNew", function(opts)
    M.new({ args = opts.args })
  end, { nargs = "?" })

  vim.api.nvim_create_user_command("TnotesRefresh", M.refresh, {})
end

return M
