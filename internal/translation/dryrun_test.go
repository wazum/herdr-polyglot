package translation_test

import (
	"context"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

func TestDryRunMarksTheDraftInsteadOfCallingAnAPI(t *testing.T) {
	t.Parallel()

	translated, err := translation.DryRun{}.Translate(context.Background(), "Bitte behebe den Test")

	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "[dry-run] Bitte behebe den Test" {
		t.Errorf("Translate returned %q, want the draft behind a dry-run marker", translated)
	}
}
