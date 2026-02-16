local pickers = require("tnotes.telescope.pickers")

return require("telescope").register_extension({
  exports = {
    tnotes = pickers.notes,
    notes = pickers.notes,
    tags = pickers.tags,
    search = pickers.search,
  },
})
