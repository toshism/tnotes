local M = {}

local pickers = require("telescope.pickers")
local finders = require("telescope.finders")
local conf = require("telescope.config").values
local actions = require("telescope.actions")
local action_state = require("telescope.actions.state")
local previewers = require("telescope.previewers")

local cli = require("tnotes.cli")
local config = require("tnotes.config")

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

-- Notes picker - fuzzy find by title
function M.notes(opts)
  opts = get_theme_opts(opts)

  local ok, notes = cli.list()
  if not ok then
    vim.notify("Failed to list notes: " .. tostring(notes), vim.log.levels.ERROR)
    return
  end

  pickers.new(opts, {
    prompt_title = "tnotes",
    finder = finders.new_table({
      results = notes,
      entry_maker = function(note)
        local tags_str = ""
        if note.tags and #note.tags > 0 then
          tags_str = " [" .. table.concat(note.tags, ", ") .. "]"
        end
        return {
          value = note,
          display = note.title .. tags_str,
          ordinal = note.title .. " " .. (note.id or "") .. " " .. table.concat(note.tags or {}, " "),
          path = note.path,
        }
      end,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = previewers.new_buffer_previewer({
      title = "Note Preview",
      define_preview = function(self, entry)
        if entry.path then
          conf.buffer_previewer_maker(entry.path, self.state.bufnr, {
            bufname = self.state.bufname,
          })
        end
      end,
    }),
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection and selection.path then
          vim.cmd("edit " .. vim.fn.fnameescape(selection.path))
        end
      end)
      return true
    end,
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

  -- search results have an .entry wrapper, unwrap for consistency
  local notes = {}
  for _, r in ipairs(results) do
    table.insert(notes, r.entry or r)
  end

  pickers.new(opts, {
    prompt_title = "tnotes [" .. project .. "]",
    finder = finders.new_table({
      results = notes,
      entry_maker = function(note)
        local tags_str = ""
        if note.tags and #note.tags > 0 then
          -- Filter out project: and path: tags for display
          local display_tags = {}
          for _, t in ipairs(note.tags) do
            if not t:match("^project:") and not t:match("^path:") then
              table.insert(display_tags, t)
            end
          end
          if #display_tags > 0 then
            tags_str = " [" .. table.concat(display_tags, ", ") .. "]"
          end
        end
        return {
          value = note,
          display = (note.title or note.id) .. tags_str,
          ordinal = (note.title or "") .. " " .. (note.id or "") .. " " .. table.concat(note.tags or {}, " "),
          path = note.path,
        }
      end,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = previewers.new_buffer_previewer({
      title = "Note Preview",
      define_preview = function(self, entry)
        if entry.path then
          conf.buffer_previewer_maker(entry.path, self.state.bufnr, {
            bufname = self.state.bufname,
          })
        end
      end,
    }),
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection and selection.path then
          vim.cmd("edit " .. vim.fn.fnameescape(selection.path))
        end
      end)
      return true
    end,
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

  -- search results have an .entry wrapper, unwrap
  local unwrapped = {}
  for _, r in ipairs(notes) do
    table.insert(unwrapped, r.entry or r)
  end

  pickers.new(opts, {
    prompt_title = "tnotes [tag: " .. tag .. "]",
    finder = finders.new_table({
      results = unwrapped,
      entry_maker = function(note)
        return {
          value = note,
          display = note.title or note.id or "untitled",
          ordinal = (note.title or "") .. " " .. (note.id or ""),
          path = note.path,
        }
      end,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = previewers.new_buffer_previewer({
      title = "Note Preview",
      define_preview = function(self, entry)
        if entry.path then
          conf.buffer_previewer_maker(entry.path, self.state.bufnr, {
            bufname = self.state.bufname,
          })
        end
      end,
    }),
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection and selection.path then
          vim.cmd("edit " .. vim.fn.fnameescape(selection.path))
        end
      end)
      return true
    end,
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

  local ok, notes = cli.search(query)
  if not ok then
    vim.notify("Failed to search notes: " .. tostring(notes), vim.log.levels.ERROR)
    return
  end

  pickers.new(opts, {
    prompt_title = "tnotes search: " .. query,
    finder = finders.new_table({
      results = notes,
      entry_maker = function(note)
        local tags_str = ""
        if note.tags and #note.tags > 0 then
          tags_str = " [" .. table.concat(note.tags, ", ") .. "]"
        end
        return {
          value = note,
          display = note.title .. tags_str,
          ordinal = note.title,
          path = note.path,
        }
      end,
    }),
    sorter = conf.generic_sorter(opts),
    previewer = previewers.new_buffer_previewer({
      title = "Note Preview",
      define_preview = function(self, entry)
        if entry.path then
          conf.buffer_previewer_maker(entry.path, self.state.bufnr, {
            bufname = self.state.bufname,
          })
        end
      end,
    }),
    attach_mappings = function(prompt_bufnr, map)
      actions.select_default:replace(function()
        actions.close(prompt_bufnr)
        local selection = action_state.get_selected_entry()
        if selection and selection.path then
          vim.cmd("edit " .. vim.fn.fnameescape(selection.path))
        end
      end)
      return true
    end,
  }):find()
end

return M
