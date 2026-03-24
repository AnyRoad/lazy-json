package tui

const helpText = `Navigation
  j/k       move
  h/l       collapse/expand or parent/child
  gg / G    top / bottom
  ]p        next parent sibling
  zR / zM   expand all / collapse all
  za        expand array elements one level
  zA        collapse array elements
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

Settings
  t         quick preview next theme
  S         open settings dialog
  up/down   move between settings rows
  left/right change the selected setting
  enter     cycle the selected setting
  s         save settings to settings.json
  persist   saved settings restore on next launch
  themes    built-ins + config themes/*.json
  paths     JSON path display can be toggled
  wrap      long strings setting affects display only
  indent    pretty save/print/copy uses selected indent

Commands
  :w        save
  :w path   save to path
  :x        save and quit
  :print    print pretty JSON and quit
  :q / :q!  quit / force quit
  :theme    quick preview next theme
  :settings open settings dialog
  :select-path P  select node by JSON path
  :copy-*   copy path/key/value/subtree/document
  :expand-all / :collapse-all
  :next-parent-sibling
  :jq EXPR  transform whole document
  :jq! EXPR transform selected subtree

Other
  prefixes  footer shows next-key menu
  ?         help
  q         quit`
