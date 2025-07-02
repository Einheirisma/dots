return {
	'stevearc/conform.nvim',
	config = function()
		require('conform').setup({
			formatters_by_ft = {
        -- stylua: ignore start
				go  = { 'gofumpt', 'golines', 'goimports-reviser' },
				lua = { 'stylua' },
				-- stylua: ignore end
				['_'] = { 'trim_newlines', 'trim_whitespace' },
			},

			format_on_save = {
				timeout_ms = 1000,
			},
		})
	end,
}
