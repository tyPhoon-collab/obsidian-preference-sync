set clipboard=unnamed

unmap <Space>

" Quick Switcher++: ファイル/通常検索
exmap switcherPlus obcommand darlal-switcher-plus:switcher-plus:open
nmap <Space><Space> :switcherPlus<CR>

" Quick Switcher++: コマンド検索
exmap switcherPlusCommands obcommand darlal-switcher-plus:switcher-plus:open-commands
nmap <Space>p :switcherPlusCommands<CR>
