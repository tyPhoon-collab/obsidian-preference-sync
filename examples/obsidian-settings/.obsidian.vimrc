set clipboard=unnamed

unmap <Space>

" Quick Switcher++: Search files
exmap switcherPlus obcommand darlal-switcher-plus:switcher-plus:open
nmap <Space><Space> :switcherPlus<CR>

" Quick Switcher++: Search commands
exmap switcherPlusCommands obcommand darlal-switcher-plus:switcher-plus:open-commands
nmap <Space>p :switcherPlusCommands<CR>

" Typewriter mode toggle
exmap typewriterToggle obcommand typewriter-mode:typewriter-toggle
nmap <Space>w :typewriterToggle<CR>
