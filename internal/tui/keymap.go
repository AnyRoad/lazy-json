package tui

const helpText = `Navigation
  j/k       move
  h/l       collapse/expand or parent/child
  gg / G    top / bottom
  ]p        next parent sibling
  zR / zM   expand all / collapse all
  / n N     search

Editing
  e         edit scalar
  E         edit subtree in $EDITOR
  a         add object field / array item
  r         rename object key
  d         delete node

Clipboard
  yp        copy JSON path
  yk        copy object key
  yv        copy value as compact JSON
  ys        copy subtree as pretty JSON
  yj        copy whole document

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
  :copy-*   copy path/key/value/subtree/document
  :expand-all / :collapse-all
  :next-parent-sibling
  :jq EXPR  transform whole document
  :jq! EXPR transform selected subtree

Other
  ?         help
  q         quit`
