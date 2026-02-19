local M = {}

local pickers = require("tnotes.telescope.pickers")

-- Register as Telescope extension
function M.register()
  local telescope = require("telescope")

  return telescope.register_extension({
    exports = {
      tnotes = pickers.notes,
      notes = pickers.notes,
      tags = pickers.tags,
      search = pickers.search,
    },
  })
end

return M
