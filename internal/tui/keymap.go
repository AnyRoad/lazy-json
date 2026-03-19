package tui

const helpText = `Navigation
  j/k       move
  h/l       collapse/expand or parent/child
  gg / G    top / bottom
  / n N     search

Editing
  e         edit scalar
  E         edit subtree in $EDITOR
  a         add object field / array item
  r         rename object key
  d         delete node

Theme
  t         quick preview next theme
  S         open theme settings dialog
  h/l       previous / next theme in settings
  s         save preview to settings.json
  persist   saved theme restores on next launch
  themes    built-ins + config themes/*.json

Commands
  :w        save
  :w path   save to path
  :x        save and quit
  :print    print JSON and quit
  :q / :q!  quit / force quit
  :theme    quick preview next theme
  :settings open theme settings dialog
  :jq EXPR  transform whole document
  :jq! EXPR transform selected subtree

Other
  ?         help
  q         quit`
