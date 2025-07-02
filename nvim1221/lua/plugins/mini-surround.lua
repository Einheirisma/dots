return {
	'echasnovski/mini.surround',
	lazy = true,
	keys = {
    -- stylua: ignore start
    { 'coa', function() require('mini.surround').add() end,     mode = { 'n', 'v' }, desc = 'Add surrounding' },
    { 'cod', function() require('mini.surround').delete() end,  mode = { 'n', 'v' }, desc = 'Delete surrounding' },
    { 'cs',  function() require('mini.surround').replace() end, mode = { 'n', 'v' }, desc = 'Replace surrounding' },
		-- stylua: ignore end
	},
	config = function()
		require('mini.surround').setup({
			mappings = {
        -- stylua: ignore start
				add            = 'coa', -- Add surrounding in Normal and Visual modes
				delete         = 'cod', -- Delete surrounding
				find           = '', -- Find surrounding (to the right)
				find_left      = '', -- Find surrounding (to the left)
				highlight      = '', -- Highlight surrounding
				replace        = 'cs', -- Replace surrounding
				update_n_lines = '', -- Update `n_lines`

				suffix_last    = '', -- Suffix to search with "prev" method
				suffix_next    = '', -- Suffix to search with "next" method
				-- stylua: ignore end
			},

			n_lines = 50,
			search_method = 'cover_or_next',
		})
	end,
}
