package integration

import "github.com/atotto/clipboard"

type Clipboard interface {
	WriteAll(text string) error
}

type SystemClipboard struct {
	WriteFunc func(text string) error
}

func (c SystemClipboard) WriteAll(text string) error {
	if c.WriteFunc != nil {
		return c.WriteFunc(text)
	}
	return clipboard.WriteAll(text)
}
