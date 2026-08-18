package translation_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

type fixedTranslator struct{ result string }

func (f fixedTranslator) Translate(context.Context, string) (string, error) {
	return f.result, nil
}

type fakeProvider struct {
	name        string
	err         error
	seenOptions translation.Options
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) New(options translation.Options) (translation.Translator, error) {
	f.seenOptions = options
	if f.err != nil {
		return nil, f.err
	}
	return fixedTranslator{result: f.name + " translated"}, nil
}

func TestTheRegistryBuildsTheRequestedProviderWithTheGivenOptions(t *testing.T) {
	t.Parallel()
	wanted := &fakeProvider{name: "wanted"}
	registry := translation.NewRegistry(&fakeProvider{name: "other"}, wanted)
	options := translation.Options{APIKey: "key-123", TargetLanguage: "EN-GB"}

	translator, err := registry.Translator("wanted", options)
	if err != nil {
		t.Fatalf("Translator returned unexpected error: %v", err)
	}
	translated, _ := translator.Translate(context.Background(), "draft")
	if translated != "wanted translated" {
		t.Errorf("got translator producing %q, want the one from the wanted provider", translated)
	}
	if wanted.seenOptions != options {
		t.Errorf("provider received options %+v, want %+v", wanted.seenOptions, options)
	}
}

func TestAnEmptyProviderNameFallsBackToTheFirstRegistered(t *testing.T) {
	t.Parallel()
	registry := translation.NewRegistry(&fakeProvider{name: "first"}, &fakeProvider{name: "second"})

	if registry.Default() != "first" {
		t.Errorf("Default is %q, want the first registered provider", registry.Default())
	}

	translator, err := registry.Translator("", translation.Options{})
	if err != nil {
		t.Fatalf("Translator returned unexpected error: %v", err)
	}
	if translated, _ := translator.Translate(context.Background(), "draft"); translated != "first translated" {
		t.Errorf("got translator producing %q, want the first provider's", translated)
	}
}

func TestAnUnknownProviderNamesTheOnesAvailable(t *testing.T) {
	t.Parallel()
	registry := translation.NewRegistry(&fakeProvider{name: "first"}, &fakeProvider{name: "second"})

	_, err := registry.Translator("nope", translation.Options{})

	if err == nil {
		t.Fatal("Translator returned no error, want a complaint about the unknown provider")
	}
	for _, name := range []string{"nope", "first", "second"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %q", err, name)
		}
	}
}

func TestAProviderThatCannotBeBuiltReportsWhy(t *testing.T) {
	t.Parallel()
	refusal := errors.New("no API key")
	registry := translation.NewRegistry(&fakeProvider{name: "picky", err: refusal})

	_, err := registry.Translator("picky", translation.Options{})

	if !errors.Is(err, refusal) {
		t.Errorf("Translator returned %v, want the provider's own error", err)
	}
}

func TestTheRegistryListsProvidersInRegistrationOrder(t *testing.T) {
	t.Parallel()
	registry := translation.NewRegistry(&fakeProvider{name: "first"}, &fakeProvider{name: "second"})

	if names := registry.Names(); !slices.Equal(names, []string{"first", "second"}) {
		t.Errorf("Names returned %v, want registration order", names)
	}
}
