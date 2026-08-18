package translation

import "context"

// DryRun stands in for a real service so the path from overlay to agent can be
// exercised without credentials or API quota.
type DryRun struct{}

func (DryRun) Translate(_ context.Context, draft string) (string, error) {
	return "[dry-run] " + draft, nil
}

type DryRunProvider struct{}

func (DryRunProvider) Name() string { return "dry-run" }

func (DryRunProvider) New(Options) (Translator, error) {
	return DryRun{}, nil
}
