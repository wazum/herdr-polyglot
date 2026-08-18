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

var ErrBlankDraft = errors.New("nothing to translate — the draft is empty")

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
	// preview translates a draft that is still being written, which happens again
	// after every pause. A translator that pays per sentence belongs here.
	preview Translator
	// There are two ways a prompt reaches an agent and no more, so they are two
	// fields rather than a map that could hold a third.
	sender Target
	typer  Target
}

type Option func(*Flow)

// WithPreviewTranslator sets what translates a draft while it is written. Without
// it, previews go the same way as a send.
func WithPreviewTranslator(previewing Translator) Option {
	return func(f *Flow) { f.preview = previewing }
}

func New(translator Translator, sending, typing Target, options ...Option) *Flow {
	flow := &Flow{
		translator: translator,
		preview:    translator,
		sender:     sending,
		typer:      typing,
	}
	for _, option := range options {
		option(flow)
	}
	return flow
}

func (f *Flow) Submit(ctx context.Context, draft string, how Delivery) (string, error) {
	translated, err := f.translate(ctx, f.translator, draft)
	if err != nil {
		return "", err
	}
	if err := f.Deliver(ctx, translated, how); err != nil {
		return "", err
	}
	return translated, nil
}

// Translate is the draft on its way to being read, not sent.
func (f *Flow) Translate(ctx context.Context, draft string) (string, error) {
	return f.translate(ctx, f.preview, draft)
}

func (f *Flow) translate(ctx context.Context, translator Translator, draft string) (string, error) {
	if strings.TrimSpace(draft) == "" {
		return "", ErrBlankDraft
	}
	translated, err := translator.Translate(ctx, draft)
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

	switch how {
	case Sending:
		return f.sender.Insert(ctx, prompt)
	case Typing:
		return f.typer.Insert(ctx, prompt)
	default:
		return fmt.Errorf("no way to deliver a prompt as %v", how)
	}
}

func (d Delivery) String() string {
	switch d {
	case Sending:
		return "sending"
	case Typing:
		return "typing"
	default:
		return fmt.Sprintf("delivery(%d)", int(d))
	}
}
