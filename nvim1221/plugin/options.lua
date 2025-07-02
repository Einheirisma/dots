-- NOTE: These 2 need to be set up before any plugins are loaded.
vim.g.mapleader = ' '
vim.g.maplocalleader = ' '

-- [[ Setting options ]]
-- See `:help vim.o`
vim.o.guicursor = ''

-- Set highlight on search
vim.o.hlsearch = true
vim.keymap.set('n', '<Esc>', '<cmd>nohlsearch<CR>')

-- Preview substitutions live, as you type!
vim.o.inccommand = 'split'

-- Minimal number of screen lines to keep above and below the cursor.
vim.o.scrolloff = 8

-- Make line numbers default
vim.wo.number = true

-- Enable mouse mode
vim.o.mouse = ''

-- Indent
vim.o.expandtab = true
vim.o.smartindent = true
vim.o.tabstop = 4
vim.o.softtabstop = 4
vim.o.shiftwidth = 4

-- Save undo history
vim.o.undofile = true

-- Case-insensitive searching UNLESS \C or capital in search
vim.o.ignorecase = true
vim.o.smartcase = true

-- Keep signcolumn on by default
vim.wo.signcolumn = 'yes'
vim.wo.relativenumber = true

-- Decrease update time
vim.o.updatetime = 50

-- Set completeopt to have a better completion experience
vim.o.completeopt = 'menu,preview,noselect'

-- NOTE: You should make sure your terminal supports this
vim.o.termguicolors = true
--
-- gets rid of line with white spaces
vim.g.editorconfig = true
vim.opt.title = true
vim.opt.titlestring = '%t%( %M%)%( (%{expand("%:~:h")})%)%a (nvim)'

vim.opt.isfname:append('@-@')
vim.opt.clipboard:append('unnamedplus') --use system clipboard as default

--vim.g.netrw_liststyle=0
--vim.g.netrw_banner=0
vim.cmd('let g:netrw_banner = 0')
