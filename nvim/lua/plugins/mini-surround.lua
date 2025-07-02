return {
	'echasnovski/mini.surround',
	lazy = true,
	keys = {
    -- stylua: ignore start
    { 'sa', function() require('mini.surround').add() end,     mode = { 'n', 'v' }, desc = 'Add surrounding' },
    { 'sd', function() require('mini.surround').delete() end,  mode = { 'n', 'v' }, desc = 'Delete surrounding' },
    { 'sr', function() require('mini.surround').replace() end, mode = { 'n', 'v' }, desc = 'Replace surrounding' },
		-- stylua: ignore end
	},

	config = function()
		require('mini.surround').setup({
			mappings = {
        -- stylua: ignore start
				add            = 'sa',
				delete         = 'sd',
				find           = '',
				find_left      = '',
				highlight      = '',
				replace        = 'sr',
				update_n_lines = '',

				suffix_last    = '',
				suffix_next    = '',
				-- stylua: ignore end
			},

			n_lines = 35,
			search_method = 'cover_or_next',
		})
	end,
}
