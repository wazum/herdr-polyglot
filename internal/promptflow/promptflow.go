// Package promptflow turns a draft written in the author's own language into an
// English prompt sitting in a coding agent's input.
package promptflow

import (
	"context"
	"errors"
	"strings"

	"github.com/wazum/herdr-polyglot/internal/translation"
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
	translated, err := f.translator.Translate(ctx, draft)
	if err != nil {
		return "", err
	}
	return readable(translated), nil
}

// The second result says whether the service keeps count at all.
func (f *Flow) Usage(ctx context.Context) (translation.Usage, bool, error) {
	reporter, keepsCount := f.translator.(translation.UsageReporter)
	if !keepsCount {
		return translation.Usage{}, false, nil
	}

	spent, err := reporter.Usage(ctx)
	if err != nil {
		return translation.Usage{}, true, err
	}
	return spent, true, nil
}

// Deliver hands over text that has already been translated, so a prompt the
// author has seen is not translated a second time on its way out.
func (f *Flow) Deliver(ctx context.Context, text string) error {
	prompt := plainText(text)
	if prompt == "" {
		return ErrBlankDraft
	}
	return f.target.Insert(ctx, prompt)
}
