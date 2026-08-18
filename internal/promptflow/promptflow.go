// Package promptflow turns a draft written in the author's own language into an
// English prompt sitting in a coding agent's input.
package promptflow

import (
	"context"
	"errors"
	"strings"
)

var ErrBlankDraft = errors.New("draft is blank")

type Translator interface {
	Translate(ctx context.Context, draft string) (string, error)
}

type Target interface {
	Insert(ctx context.Context, text string) error
}

type Flow struct {
	translator Translator
	target     Target
}

func New(translator Translator, target Target) *Flow {
	return &Flow{translator: translator, target: target}
}

func (f *Flow) Submit(ctx context.Context, draft string) (string, error) {
	if strings.TrimSpace(draft) == "" {
		return "", ErrBlankDraft
	}

	translated, err := f.translator.Translate(ctx, draft)
	if err != nil {
		return "", err
	}
	if err := f.target.Insert(ctx, translated); err != nil {
		return "", err
	}
	return translated, nil
}
