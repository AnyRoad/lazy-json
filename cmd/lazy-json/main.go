package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anyroad/lazy-json/internal/document"
	"github.com/anyroad/lazy-json/internal/source"
	"github.com/anyroad/lazy-json/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	input, err := source.Load(args, os.Stdin)
	if err != nil {
		if errors.Is(err, source.ErrNoInput) {
			return fmt.Errorf("usage: lazy-json <file.json> or cat file.json | lazy-json")
		}
		return err
	}

	doc, err := document.Parse(input.Data)
	if err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	model := tui.NewModel(doc, input)
	opts := []tea.ProgramOption{}
	if input.Kind == source.KindStdin {
		opts = append(opts, tea.WithInputTTY())
	}
	program := tea.NewProgram(model, opts...)
	finalModel, err := program.Run()
	if err != nil {
		return err
	}
	typed, ok := finalModel.(*tui.Model)
	if !ok {
		return nil
	}
	if len(typed.ExitOutput) > 0 {
		if _, err := os.Stdout.Write(typed.ExitOutput); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
	}
	return nil
}
