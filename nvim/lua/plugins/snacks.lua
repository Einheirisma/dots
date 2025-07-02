return {
	'folke/snacks.nvim',

	dependencies = { 'nvim-tree/nvim-web-devicons' },
	lazy = false,
	keys = {
    -- stylua: ignore start
    { '<Leader><Space>', function() require('snacks').picker.files() end,                                 desc = 'Find Files' },
    { '<Leader>/',       function() require('snacks').picker.grep() end,                                  desc = 'Grep' },
    { '<Leader>sW',      function() require('snacks').picker.grep_word() end,       mode = { 'n', 'x' },  desc = 'Visual selection or word', },
    { '<Leader>sC',      function() require('snacks').picker.colorschemes() end,                          desc = 'Colorschemes' },
    { '<Leader>sD',      function() require('snacks').picker.diagnostics() end,                           desc = 'Diagnostics' },
    { '<Leader>zD',      function() require('snacks').picker.pickers() end,                           desc = 'Diagnostics' },
    { '<Leader>sH',      function() require('snacks').picker.command_history() end,                       desc = 'Command History' },
    { '<Leader>sI',      function() require('snacks').picker.icons() end,                                 desc = 'Icons' },
    { '<Leader>sK',      function() require('snacks').picker.keymaps() end,                               desc = 'Keymaps' },
    { '<Leader>sN',      function() require('snacks').picker.notifications() end,                         desc = 'Notification History' },
    { '<Leader>sR',      function() require('snacks').picker.recent() end,                                desc = 'Recent' },
    { '<Leader>sU',      function() require('snacks').picker.undo() end,                                  desc = 'Undo History' },
    --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --  --
    { '<Leader>renf',    function() require('snacks').rename.rename_file() end,                           desc = 'Rename current file' },
		-- stylua: ignore end
	},

	config = function()
		vim.g.snacks_animate = false

		require('snacks').setup({
			-- animate = {},
			bigfile = {
				enabled = true,
				notify = true, -- show notification when big file detected
				size = 1.5 * 1024 * 1024, -- 1.5MB
				line_length = 1000, -- average line length (useful for minified files)
				-- Enable or disable features when big file detected
				setup = function(ctx)
					if vim.fn.exists(':NoMatchParen') ~= 0 then
						vim.cmd([[NoMatchParen]])
					end
					require('snacks').util.wo(0, { foldmethod = 'manual', statuscolumn = '', conceallevel = 0 })
					vim.b.minianimate_disable = true
					vim.schedule(function()
						if vim.api.nvim_buf_is_valid(ctx.buf) then
							vim.bo[ctx.buf].syntax = ctx.ft
						end
					end)
				end,
			},
			-- bufdelete = {},
			dashboard = { enabled = true },
			-- debug = {},
			-- dim = {},
			-- explorer = {},
			-- git = {},
			-- gitbrowse = {},
			image = { enabled = true },
			-- indent = {},
			input = { enabled = true },
			-- layout = {},
			-- lazygit = { enabled = true },
			notifier = {
				enabled = true,
				level = vim.log.levels.TRACE,
				style = 'fancy',
			},
			notify = { enabled = true },

			picker = { enabled = true },

			-- profiler = {},
			-- quickfile = {},
			rename = { enabled = true },
			-- scope = {}, но интересно
			-- scratch = {},
			-- scroll = {},
			statuscolumn = { enabled = true },
			-- terminal = {},
			-- toggle = {},
			-- util = {},
			-- win = {},
			-- words = {},
			-- zen = {},
		})
	end,
}
