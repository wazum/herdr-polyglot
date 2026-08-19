package main

import (
	"context"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/config"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

func TestWithoutAKeyThePopupStillOpensAsAPlainDraftBox(t *testing.T) {
	chosen := chooseService(&config.Settings{})

	if chosen.name != "off" {
		t.Errorf("without a key the service is %q, want it off", chosen.name)
	}
	if chosen.trouble != nil {
		t.Errorf("without a key there is trouble to report: %v", chosen.trouble)
	}
	if chosen.translates {
		t.Error("without a key the popup claims it translates")
	}

	written := "Bitte behebe den fehlschlagenden Test"
	delivered, err := chosen.translator.Translate(context.Background(), written)
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if delivered != written {
		t.Errorf("Translate returned %q, want the draft as it was written", delivered)
	}
}

func TestAKeyOnItsOwnPicksTheDefaultService(t *testing.T) {
	chosen := chooseService(&config.Settings{
		Options: translation.Options{APIKey: "key-123", TargetLanguage: "EN-US"},
	})

	if chosen.name != "deepl" {
		t.Errorf("with a key the service is %q, want the default", chosen.name)
	}
	if !chosen.translates {
		t.Error("with a key the popup does not claim to translate")
	}
	if chosen.trouble != nil {
		t.Errorf("with a key there is trouble to report: %v", chosen.trouble)
	}
}

// Asking for a service that cannot be built is not the same as asking for none:
// the draft must never reach an agent untranslated because a key was missing.
func TestAServiceThatCannotBeBuiltSaysSoAndRefusesToTranslate(t *testing.T) {
	chosen := chooseService(&config.Settings{
		Provider:   "deepl",
		ConfigFile: "/somewhere/.env",
	})

	if chosen.trouble == nil {
		t.Fatal("a service that cannot be built reports no trouble")
	}
	if !strings.Contains(chosen.trouble.Error(), "/somewhere/.env") {
		t.Errorf("the trouble is %v, want it to say where the key belongs", chosen.trouble)
	}
	if !chosen.translates {
		t.Error("the popup pretends translation is off when the service is only broken")
	}

	if _, err := chosen.translator.Translate(context.Background(), "Bitte behebe es"); err == nil {
		t.Error("the broken service translated something instead of saying why it cannot")
	}
}

// A command needs no key, so setting one is enough to choose it: there is nothing
// else it could mean.
func TestACommandOnItsOwnPicksTheLocalService(t *testing.T) {
	chosen := chooseService(&config.Settings{
		Options: translation.Options{Command: "sed 's/behebe/fix/'", TargetLanguage: "EN-US"},
	})

	if chosen.name != "cmd" {
		t.Errorf("with a command the service is %q, want cmd", chosen.name)
	}
	if !chosen.translates {
		t.Error("with a command the popup does not claim to translate")
	}
	if chosen.trouble != nil {
		t.Errorf("with a command there is trouble to report: %v", chosen.trouble)
	}

	translated, err := chosen.translator.Translate(context.Background(), "Bitte behebe es")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "Bitte fix es" {
		t.Errorf("Translate returned %q, want what the command answered", translated)
	}
}

// A key and a command together say nothing about which was meant, and guessing
// wrong would send a draft to a service the person thought they had left behind.
func TestAKeyAndACommandTogetherAreRefusedRatherThanGuessedAt(t *testing.T) {
	chosen := chooseService(&config.Settings{
		Options: translation.Options{APIKey: "key-123", Command: "cat"},
	})

	if chosen.trouble == nil {
		t.Fatal("a key and a command together report no trouble")
	}
	if !strings.Contains(chosen.trouble.Error(), "HERDR_POLYGLOT_PROVIDER") {
		t.Errorf("the trouble is %v, want it to say which setting decides", chosen.trouble)
	}
	if _, err := chosen.translator.Translate(context.Background(), "Bitte behebe es"); err == nil {
		t.Error("something was translated while it is unclear by what")
	}
}

func TestAnExplicitServiceIsUsedAsAsked(t *testing.T) {
	chosen := chooseService(&config.Settings{Provider: "dry-run"})

	if chosen.name != "dry-run" {
		t.Errorf("the service is %q, want the one asked for", chosen.name)
	}
	translated, err := chosen.translator.Translate(context.Background(), "Bitte behebe es")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if !strings.Contains(translated, "dry-run") {
		t.Errorf("Translate returned %q, want the dry-run marker", translated)
	}
}
