local M = {}

local config = require("tnotes.config")
local commands = require("tnotes.commands")
local cli = require("tnotes.cli")

-- Check if tnotes binary is available
local function check_binary()
  local handle = io.popen(config.options.bin .. " --version 2>&1")
  if handle then
    local result = handle:read("*a")
    handle:close()
    return result ~= nil and result ~= ""
  end
  return false
end

-- Setup keymappings
local function setup_mappings()
  local mappings = config.options.mappings

  if mappings.follow_link then
    vim.keymap.set("n", mappings.follow_link, function()
      -- Only override in markdown files
      local ft = vim.bo.filetype
      if ft == "markdown" then
        commands.follow_link()
      else
        vim.cmd("normal! gf")
      end
    end, { desc = "Follow tnotes link or default gf" })
  end

  if mappings.new_note then
    vim.keymap.set("n", mappings.new_note, function()
      commands.new()
    end, { desc = "Create new tnotes note" })
  end

  if mappings.search then
    vim.keymap.set("n", mappings.search, function()
      require("tnotes.telescope.pickers").notes()
    end, { desc = "Search tnotes" })
  end

  if mappings.tags then
    vim.keymap.set("n", mappings.tags, function()
      require("tnotes.telescope.pickers").tags()
    end, { desc = "Browse tnotes tags" })
  end
end

-- Main setup function
function M.setup(opts)
  config.setup(opts)

  -- Register commands
  commands.setup()

  -- Setup keymappings
  setup_mappings()

  -- Load telescope extension
  pcall(function()
    require("telescope").load_extension("tnotes")
  end)

  -- Check binary availability (non-blocking warning)
  vim.defer_fn(function()
    if not check_binary() then
      vim.notify(
        "tnotes: binary not found. Install tnotes or set bin option.",
        vim.log.levels.WARN
      )
    end
  end, 100)
end

-- Expose CLI functions for advanced usage
M.cli = cli

-- Expose telescope pickers
M.pickers = require("tnotes.telescope.pickers")

return M
