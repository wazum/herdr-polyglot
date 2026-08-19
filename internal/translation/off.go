package translation

import "context"

// Off is no translation at all: the draft reaches the agent as it was written.
// With no service configured the popup is still a draft box worth having, and this
// is what stands in the place of a translator.
type Off struct{}

func (Off) Translate(_ context.Context, draft string) (string, error) { return draft, nil }

type OffProvider struct{}

func (OffProvider) Name() string { return "off" }

func (OffProvider) New(Options) (Translator, error) { return Off{}, nil }

// Broken is a service that was asked for and could not be built. It keeps the
// popup usable while refusing to translate, so a draft is never delivered in the
// language it was written in because a key was missing.
type Broken struct{ Why error }

func (b Broken) Translate(context.Context, string) (string, error) { return "", b.Why }
