return {
	'echasnovski/mini.files',
	lazy = true,
	dependencies = { 'nvim-tree/nvim-web-devicons' },
  -- stylua: ignore start
  keys = {{ '<Leader>E', function() require('mini.files').open() end, desc = 'Open Mini.files' }},
	-- stylua: ignore end
	config = function()
		require('mini.files').setup({
			mappings = {
				close = '<ESC>',
				go_in = 'l',
				go_in_plus = 'L',
				go_out = 'h',
				go_out_plus = 'H',
				mark_goto = "'",
				mark_set = 'm',
				reset = ',',
				reveal_cwd = '.',
				show_help = 'g?',
				synchronize = 'S',
				trim_left = '<',
				trim_right = '>',
			},

			options = {
				permanent_delete = false,
			},
		})
	end,
}
