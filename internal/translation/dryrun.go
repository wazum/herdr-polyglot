// Package translation provides the translators the prompt flow can run on.
package translation

import "context"

// DryRun stands in for a real translator so the plumbing between overlay and
// agent can be exercised without spending API quota.
type DryRun struct{}

func (DryRun) Translate(_ context.Context, draft string) (string, error) {
	return "[dry-run] " + draft, nil
}
