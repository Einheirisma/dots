-- Keymaps for better default experience
-- See `:help vim.keymap.set()`
-- stylua: ignore start
vim.keymap.set('v', 'J', ":m '>+1<CR>gv=gv", { desc = 'Moves Line Down' })
vim.keymap.set('v', 'K', ":m '<-2<CR>gv=gv", { desc = 'Moves Line Up' })
vim.keymap.set('n', '<C-d>', '<C-d>zz', { desc = 'Scroll Down' })
vim.keymap.set('n', '<C-u>', '<C-u>zz', { desc = 'Scroll Up' })
vim.keymap.set('n', 'n', 'nzzzv', { desc = 'Next Search Result' })
vim.keymap.set('n', 'N', 'Nzzzv', { desc = 'Previous Search Result' })

-- see help sticky keys on windows
vim.cmd([[command! W w]])
vim.cmd([[command! Wq wq]])
vim.cmd([[command! WQ wq]])
vim.cmd([[command! Q q]])

-- Remap for dealing with word wrap
vim.keymap.set('n', 'k', 'v:count == 0 ? "gk" : "k"', { expr = true, silent = true })
vim.keymap.set('n', 'j', 'v:count == 0 ? "gj" : "j"', { expr = true, silent = true })

-- You should instead use these keybindings so that they are still easy to use, but dont conflict
vim.keymap.set('n', '<leader>y', '"+y', { noremap = true, silent = true, desc = 'Yank to clipboard' })
vim.keymap.set({'v', 'x'}, '<leader>y', '"+y', { noremap = true, silent = true, desc = 'Yank to clipboard' })
vim.keymap.set({'n', 'v', 'x'}, '<leader>yy', '"+yy', { noremap = true, silent = true, desc = 'Yank line to clipboard' })
vim.keymap.set({'n', 'v', 'x'}, '<leader>Y', '"+yy', { noremap = true, silent = true, desc = 'Yank line to clipboard' })
vim.keymap.set({'n', 'v', 'x'}, '<C-a>', 'gg0vG$', { noremap = true, silent = true, desc = 'Select all' })
vim.keymap.set({'n', 'v', 'x'}, '<leader>p', '"+p', { noremap = true, silent = true, desc = 'Paste from clipboard' })
vim.keymap.set('i', '<C-p>', '<C-r><C-p>+', { noremap = true, silent = true, desc = 'Paste from clipboard from within insert mode' })
vim.keymap.set('x', '<leader>P', '"_dP', { noremap = true, silent = true, desc = 'Paste over selection without erasing unnamed register' })
-- stylua: ignore end
