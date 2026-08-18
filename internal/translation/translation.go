// Package translation defines what a translation service must offer and keeps
// the registry of the services this build knows about.
package translation

import (
	"context"
	"errors"
)

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

type Usage struct {
	Used  int64
	Limit int64
}

// Not every service keeps count, so this is asked for rather than required.
type UsageReporter interface {
	Usage(ctx context.Context) (Usage, error)
}

// ErrNoUsage says the service does not report what it has spent.
var ErrNoUsage = errors.New("the service keeps no count")

// ReporterOf looks through anything that wraps a translator, so a service behind
// a cache still reports its allowance.
func ReporterOf(translator Translator) (UsageReporter, error) {
	for translator != nil {
		if reporter, keepsCount := translator.(UsageReporter); keepsCount {
			return reporter, nil
		}

		wrapper, wraps := translator.(interface{ Unwrap() Translator })
		if !wraps {
			break
		}
		translator = wrapper.Unwrap()
	}
	return nil, ErrNoUsage
}
