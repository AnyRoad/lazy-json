package integration

import (
	"errors"
	"testing"

	"github.com/anyroad/lazy-json/internal/document"
)

type stubRunner struct {
	stdout []byte
	stderr []byte
	err    error
}

func (s stubRunner) Run(_ string, _ []string, _ []byte) ([]byte, []byte, error) {
	return s.stdout, s.stderr, s.err
}

func TestApplyJQSuccess(t *testing.T) {
	node, err := document.ParseNode([]byte(`{"count":1}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyJQ(stubRunner{stdout: []byte(`{"count":2}`)}, ".count = 2", node)
	if err != nil {
		t.Fatalf("ApplyJQ() error = %v", err)
	}
	if got.Object[0].Value.Number != "2" {
		t.Fatalf("count = %q", got.Object[0].Value.Number)
	}
}

func TestApplyJQInvalidOutput(t *testing.T) {
	node, err := document.ParseNode([]byte(`{"count":1}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyJQ(stubRunner{stdout: []byte(`1 2`)}, ".", node)
	if err == nil {
		t.Fatal("ApplyJQ() error = nil, want error")
	}
}

func TestApplyJQFailure(t *testing.T) {
	node, err := document.ParseNode([]byte(`{"count":1}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyJQ(stubRunner{
		stderr: []byte("jq: parse error"),
		err:    errors.New("exit status 3"),
	}, ".", node)
	if err == nil {
		t.Fatal("ApplyJQ() error = nil, want error")
	}
}
