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
	translated, err := f.Translate(ctx, draft)
	if err != nil {
		return "", err
	}
	if err := f.Deliver(ctx, translated); err != nil {
		return "", err
	}
	return translated, nil
}

func (f *Flow) Translate(ctx context.Context, draft string) (string, error) {
	if strings.TrimSpace(draft) == "" {
		return "", ErrBlankDraft
	}
	return f.translator.Translate(ctx, draft)
}

// Deliver hands over text that has already been translated, so a prompt the
// author has seen is not translated a second time on its way out.
func (f *Flow) Deliver(ctx context.Context, text string) error {
	if strings.TrimSpace(text) == "" {
		return ErrBlankDraft
	}
	return f.target.Insert(ctx, text)
}
