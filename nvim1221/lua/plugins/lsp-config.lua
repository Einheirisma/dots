return {
	{
		'williamboman/mason.nvim',
		config = function()
			require('mason').setup()
		end,
	},
	{
		'williamboman/mason-lspconfig.nvim',
		config = function()
			require('mason-lspconfig').setup({
				ensure_installed = {
					'bashls',
					'tinymist',
					'typos_lsp',
					'gopls',
					'lua_ls',
				},
			})
		end,
	},
	{
		'WhoIsSethDaniel/mason-tool-installer',
		config = function()
			require('mason-tool-installer').setup({
				ensure_installed = {
					'gofumpt',
					'goimports-reviser',
					'golangci-lint',
					'stylua',
				},
			})
		end,
	},
	{
		'neovim/nvim-lspconfig',
		config = function()
			require('lspconfig').bashls.setup({
				cmd = { 'bash-language-server', 'start' },
				filetypes = { 'bash', 'sh' },
				root_markers = { '.git' },
				settings = {
					bashIde = {
						globPattern = '*@(.sh|.inc|.bash|.command)',
					},
				},
			})

			require('lspconfig').gopls.setup({
				filetypes = { 'go', 'gomod', 'gowork', 'gotmpl' },
			})

			require('lspconfig').lua_ls.setup({
				filetypes = { 'lua' },
				settings = {
					Lua = {
						completion = {
							callSnippet = 'Replace',
							displayContext = 1,
						},
						diagnostics = {
							globals = { 'vim' },
						},
						hint = {
							enable = true,
							arrayIndex = 'Enable',
						},
						runtime = {
							version = 'LuaJIT',
						},
						telemetry = {
							enabled = false,
						},
					},
				},
			})

			require('lspconfig').tinymist.setup({
				cmd = { 'tinymist' },
				filetypes = { 'typst' },
				root_markers = { '.git' },
			})

			require('lspconfig').typos_lsp.setup({
				cmd = { 'typos-lsp' },
				root_markers = { 'typos.toml', '_typos.toml', '.typos.toml', 'pyproject.toml', 'Cargo.toml' },
				settings = {},
			})
		end,
	},
}
