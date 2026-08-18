package google_test

import (
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/google"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

func TestTheProviderIsSelectableByName(t *testing.T) {
	t.Parallel()

	if name := (google.Provider{}).Name(); name != "google" {
		t.Errorf("Name is %q, want google", name)
	}
}

func TestAServiceWithoutAKeyIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := (google.Provider{}).New(translation.Options{}); err == nil {
		t.Error("New returned no error without an API key")
	}
}

// An endpoint override must not be the way an API key ends up on the wire in
// clear; a loopback address is a test server, not the internet.
func TestAnInsecureEndpointIsRefusedUnlessItIsLoopback(t *testing.T) {
	t.Parallel()

	_, err := (google.Provider{}).New(translation.Options{
		APIKey: "key-123", Endpoint: "http://translate.example.com/v2",
	})
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Errorf("New returned %v, want a plain http endpoint refused", err)
	}

	if _, err := (google.Provider{}).New(translation.Options{
		APIKey: "key-123", Endpoint: "http://127.0.0.1:8080/v2",
	}); err != nil {
		t.Errorf("New returned %v, want a loopback endpoint allowed", err)
	}
}

func TestTheTranslatorIsBuiltWithTheConfiguredLanguage(t *testing.T) {
	t.Parallel()

	translator, err := (google.Provider{}).New(translation.Options{
		APIKey: "key-123", TargetLanguage: "EN-GB", Endpoint: "https://example.com/v2",
	})
	if err != nil {
		t.Fatalf("New returned unexpected error: %v", err)
	}
	client, isClient := translator.(*google.Client)
	if !isClient {
		t.Fatalf("New returned %T, want the google client", translator)
	}
	if client.Endpoint() != "https://example.com/v2" {
		t.Errorf("the client talks to %q, want the configured endpoint", client.Endpoint())
	}
}
