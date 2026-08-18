// Package translation defines what a translation service must offer and keeps
// the registry of the services this build knows about.
package translation

import "context"

type Translator interface {
	Translate(ctx context.Context, draft string) (string, error)
}

// Options is the superset of settings; each service takes what applies and
// rejects what it cannot work without.
type Options struct {
	APIKey         string
	TargetLanguage string
	Endpoint       string
}

type Provider interface {
	Name() string
	New(Options) (Translator, error)
}
