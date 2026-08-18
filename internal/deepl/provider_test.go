package deepl_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/deepl"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

func TestTheProviderIsRegisteredUnderItsServiceName(t *testing.T) {
	t.Parallel()

	if name := (deepl.Provider{}).Name(); name != "deepl" {
		t.Errorf("Name is %q, want deepl", name)
	}
}

func TestTheProviderRefusesToStartWithoutAnApiKey(t *testing.T) {
	t.Parallel()

	_, err := deepl.Provider{}.New(translation.Options{TargetLanguage: "EN-US"})

	if err == nil {
		t.Fatal("New returned no error, want a complaint about the missing key")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("error %q does not mention the missing API key", err)
	}
}

func TestTheProviderBuildsATranslatorForTheGivenOptions(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK,
		`{"translations":[{"text":"Please fix the failing test"}]}`)

	translator, err := deepl.Provider{}.New(translation.Options{
		APIKey:         "key-123",
		TargetLanguage: "EN-GB",
		Endpoint:       server.URL,
	})
	if err != nil {
		t.Fatalf("New returned unexpected error: %v", err)
	}

	translated, err := translator.Translate(context.Background(), "Bitte behebe den fehlschlagenden Test")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "Please fix the failing test" {
		t.Errorf("Translate returned %q, want the translated text", translated)
	}
	if captured.body["target_lang"] != "EN-GB" {
		t.Errorf("request asked for %v, want the language from the options", captured.body["target_lang"])
	}
	if captured.authorization != "DeepL-Auth-Key key-123" {
		t.Errorf("request carried authorization %q, want the key from the options", captured.authorization)
	}
}
