-- [[ Highlight on yank ]]
-- See `:help vim.highlight.on_yank()`
local highlight_group = vim.api.nvim_create_augroup('YankHighlight', { clear = true })
vim.api.nvim_create_autocmd('TextYankPost', {
	callback = function()
		vim.highlight.on_yank()
	end,
	group = highlight_group,
	pattern = '*',
})

-- [[ Disable autocomment on Enter ]]
-- See :help formatoptions
vim.api.nvim_create_autocmd('FileType', {
	desc = 'remove formatoptions',
	callback = function()
		vim.opt.formatoptions:remove({ 'c', 'r', 'o' })
	end,
})
