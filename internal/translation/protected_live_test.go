//go:build live

package translation_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/config"
	"github.com/wazum/herdr-polyglot/internal/deepl"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

// The prompts a service was rewriting, put through the real one with protection:
//
//	HERDR_PLUGIN_CONFIG_DIR="$(herdr plugin config-dir wazum.polyglot)" \
//	HERDR_POLYGLOT_TARGET=w1:p1 go test -tags live -run LiveProtection -v ./internal/translation/
func TestLiveProtectionKeepsCodeAsItWas(t *testing.T) {
	settings, err := config.Load(os.Getenv)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if settings.Options.APIKey == "" {
		t.Skip("no API key configured")
	}
	service, err := deepl.Provider{}.New(settings.Options)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	translator := translation.Protecting(translation.Segmented(service))

	block := "```go\nif kunde == nil { // sollte nie passieren\n\treturn fmt.Errorf(\"kunde fehlt\")\n}\n```"
	for _, wanted := range []struct{ draft, kept string }{
		{
			"Schreibe einen Test für `berechneRabatt(warenkorb, prozent)` bitte.",
			"`berechneRabatt(warenkorb, prozent)`",
		},
		{"Der Fehler steht hier:\n\n" + block + "\n\nBitte behebe das gründlich.", block},
		{"Ersetze `alterName` durch `neuerName` und lass alles andere.", "`alterName`"},
	} {
		english, err := translator.Translate(context.Background(), wanted.draft)
		if err != nil {
			t.Errorf("Translate returned: %v", err)
			continue
		}
		if !strings.Contains(english, wanted.kept) {
			t.Errorf("the protected part did not come back:\n  want: %s\n  got:  %s", wanted.kept, english)
		}
		t.Logf("\n  DE: %s\n  EN: %s", wanted.draft, english)
	}
}
