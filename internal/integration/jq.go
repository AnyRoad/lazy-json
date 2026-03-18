package integration

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/andrei/lazy-json/internal/document"
)

type JQRunner interface {
	Run(name string, args []string, stdin []byte) (stdout []byte, stderr []byte, err error)
}

type ExecJQRunner struct{}

func (ExecJQRunner) Run(name string, args []string, stdin []byte) ([]byte, []byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func ApplyJQ(runner JQRunner, expr string, node *document.Node) (*document.Node, error) {
	if runner == nil {
		runner = ExecJQRunner{}
	}
	stdin, err := document.MarshalCompactNode(node)
	if err != nil {
		return nil, fmt.Errorf("serialize json for jq: %w", err)
	}
	stdout, stderr, err := runner.Run("jq", []string{expr}, stdin)
	if err != nil {
		if len(stderr) > 0 {
			return nil, fmt.Errorf("jq failed: %s", bytes.TrimSpace(stderr))
		}
		return nil, fmt.Errorf("jq failed: %w", err)
	}
	parsed, err := document.Parse(stdout)
	if err != nil {
		return nil, fmt.Errorf("jq output is not valid single JSON value: %w", err)
	}
	return parsed.Root, nil
}
