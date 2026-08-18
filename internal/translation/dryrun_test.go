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

func TestTheDryRunProviderNeedsNoCredentials(t *testing.T) {
	t.Parallel()
	provider := translation.DryRunProvider{}

	if provider.Name() != "dry-run" {
		t.Errorf("Name is %q, want dry-run", provider.Name())
	}

	translator, err := provider.New(translation.Options{})
	if err != nil {
		t.Fatalf("New returned unexpected error: %v", err)
	}
	if translated, _ := translator.Translate(context.Background(), "Test"); translated != "[dry-run] Test" {
		t.Errorf("provider built a translator returning %q, want the marked draft", translated)
	}
}
