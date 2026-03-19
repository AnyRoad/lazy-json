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
  t         switch to the next theme preview
  S         open theme settings dialog
  h/l       previous / next theme in settings
  s         save theme setting in settings
  esc       close settings without saving

Commands
  :w        save
  :w path   save to path
  :x        save and quit
  :print    print JSON and quit
  :q / :q!  quit / force quit
  :theme    switch to the next theme preview
  :settings open theme settings dialog
  :jq EXPR  transform whole document
  :jq! EXPR transform selected subtree

Other
  ?         help
  q         quit`
