// Package translation defines what a translation service must offer and keeps
// the registry of the services this build knows about.
package translation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
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
	// Command is a shell command line that translates what it is given on its
	// input, for a service that is a program on the machine rather than a host.
	Command string
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

// Trouble says in a sentence what went wrong on the way to a service. A transport
// error carries a URL, a Go type and a method name, none of which help the person
// looking at a popup.
func Trouble(service string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return err
	case errors.Is(err, context.DeadlineExceeded), os.IsTimeout(err):
		return fmt.Errorf("%s did not answer in time", service)
	default:
		var dns *net.DNSError
		if errors.As(err, &dns) {
			return fmt.Errorf("%s could not be found — is there a network?", service)
		}
		var connecting *net.OpError
		if errors.As(err, &connecting) {
			return fmt.Errorf("%s could not be reached", service)
		}
		return fmt.Errorf("%s could not be asked: %w", service, err)
	}
}
