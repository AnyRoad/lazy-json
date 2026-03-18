package tui

import "github.com/charmbracelet/bubbles/textinput"

type promptKind string

const (
	promptNone       promptKind = ""
	promptCommand    promptKind = "command"
	promptSearch     promptKind = "search"
	promptEditScalar promptKind = "edit-scalar"
	promptRenameKey  promptKind = "rename-key"
	promptAddObject  promptKind = "add-object"
	promptAddArray   promptKind = "add-array"
)

func newPrompt() textinput.Model {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 0
	input.Width = 48
	return input
}
