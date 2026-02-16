-- tnotes.nvim - AI-native note-taking for Neovim
-- Requires: tnotes CLI, telescope.nvim, plenary.nvim

if vim.g.loaded_tnotes then
  return
end
vim.g.loaded_tnotes = true

-- Plugin is lazy-loaded via require('tnotes').setup()
-- This file just prevents double-loading
