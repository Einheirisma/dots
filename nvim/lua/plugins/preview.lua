return {
	'sylvanfranklin/omni-preview.nvim',
	dependencies = { { 'chomosuke/typst-preview.nvim', lazy = true } },
	keys = {
		{ '<Leader>sp', '<cmd>OmniPreview start<CR>', desc = 'Start OmniPreview' },
	},

	config = function()
		require('omni-preview').setup()
		require('typst-preview').setup({
			dependencies_bin = { ['tinymist'] = 'tinymist' },
		})
	end,
}
