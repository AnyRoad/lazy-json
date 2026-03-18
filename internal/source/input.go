package source

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrNoInput        = errors.New("no input provided")
	ErrAmbiguousInput = errors.New("file path and stdin cannot be used together")
)

type Kind string

const (
	KindFile  Kind = "file"
	KindStdin Kind = "stdin"
)

type Input struct {
	Kind Kind
	Path string
	Data []byte
}

func Load(args []string, stdin *os.File) (Input, error) {
	hasStdin, err := HasPipedStdin(stdin)
	if err != nil {
		return Input{}, fmt.Errorf("inspect stdin: %w", err)
	}
	if len(args) > 1 {
		return Input{}, fmt.Errorf("expected at most one file path")
	}
	if len(args) == 1 && hasStdin {
		return Input{}, ErrAmbiguousInput
	}
	if len(args) == 1 {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return Input{}, fmt.Errorf("read %q: %w", args[0], err)
		}
		return Input{Kind: KindFile, Path: args[0], Data: data}, nil
	}
	if hasStdin {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return Input{}, fmt.Errorf("read stdin: %w", err)
		}
		return Input{Kind: KindStdin, Data: data}, nil
	}
	return Input{}, ErrNoInput
}

func HasPipedStdin(stdin *os.File) (bool, error) {
	info, err := stdin.Stat()
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeCharDevice == 0, nil
}
