local M = {}

M.defaults = {
  -- Path to tnotes binary (default: searches PATH)
  bin = "tnotes",

  -- Key mappings (set to false to disable)
  mappings = {
    follow_link = "gf",
    backlinks = "<leader>nb",
    new_note = "<leader>nn",
    search = "<leader>ns",
    tags = "<leader>nt",
    insert_link = "<leader>nl",
  },

  -- Telescope options
  telescope = {
    theme = nil, -- use default, or "dropdown", "ivy", "cursor"
  },
}

M.options = {}

function M.setup(opts)
  M.options = vim.tbl_deep_extend("force", {}, M.defaults, opts or {})
end

return M
