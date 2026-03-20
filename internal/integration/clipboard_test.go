package integration

import (
	"errors"
	"testing"
)

func TestSystemClipboardUsesInjectedWriteFunc(t *testing.T) {
	var got string
	clipboard := SystemClipboard{
		WriteFunc: func(text string) error {
			got = text
			return nil
		},
	}

	if err := clipboard.WriteAll("hello"); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("WriteAll() wrote %q, want %q", got, "hello")
	}
}

func TestSystemClipboardPropagatesInjectedError(t *testing.T) {
	wantErr := errors.New("boom")
	clipboard := SystemClipboard{
		WriteFunc: func(text string) error {
			return wantErr
		},
	}

	if err := clipboard.WriteAll("hello"); !errors.Is(err, wantErr) {
		t.Fatalf("WriteAll() error = %v, want %v", err, wantErr)
	}
}
