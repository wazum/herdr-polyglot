// Package promptflow turns a draft written in the author's own language into an
// English prompt sitting in a coding agent's input.
package promptflow

import (
	"context"
	"errors"
	"fmt"
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

// Delivery is how a finished prompt reaches the agent: handed over and sent, or
// only typed into its input for the author to send.
type Delivery int

const (
	Sending Delivery = iota
	Typing
)

type Flow struct {
	translator Translator
	targets    map[Delivery]Target
}

func New(translator Translator, sending, typing Target) *Flow {
	return &Flow{
		translator: translator,
		targets:    map[Delivery]Target{Sending: sending, Typing: typing},
	}
}

func (f *Flow) Submit(ctx context.Context, draft string, how Delivery) (string, error) {
	translated, err := f.Translate(ctx, draft)
	if err != nil {
		return "", err
	}
	if err := f.Deliver(ctx, translated, how); err != nil {
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
	reporter, err := translation.ReporterOf(f.translator)
	if errors.Is(err, translation.ErrNoUsage) {
		return translation.Usage{}, false, nil
	}
	if err != nil {
		return translation.Usage{}, false, err
	}

	spent, err := reporter.Usage(ctx)
	if err != nil {
		return translation.Usage{}, true, err
	}
	return spent, true, nil
}

// Deliver hands over text that has already been translated, so a prompt the
// author has seen is not translated a second time on its way out.
func (f *Flow) Deliver(ctx context.Context, text string, how Delivery) error {
	prompt := plainText(text)
	if prompt == "" {
		return ErrBlankDraft
	}

	target, known := f.targets[how]
	if !known {
		return fmt.Errorf("no way to deliver a prompt as %v", how)
	}
	return target.Insert(ctx, prompt)
}

func (d Delivery) String() string {
	if d == Typing {
		return "typing"
	}
	return "sending"
}
