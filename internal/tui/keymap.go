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

Commands
  :w        save
  :w path   save to path
  :x        save and quit
  :print    print JSON and quit
  :q / :q!  quit / force quit
  :jq EXPR  transform whole document
  :jq! EXPR transform selected subtree

Other
  t         switch theme
  ?         help
  q         quit`
