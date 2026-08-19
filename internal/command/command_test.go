package command_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wazum/herdr-polyglot/internal/command"
)

func TestTheDraftIsWrittenToTheCommandAndItsAnswerRead(t *testing.T) {
	translator := command.New("sed 's/behebe/fix/'")

	translated, err := translator.Translate(context.Background(), "Bitte behebe den Test")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "Bitte fix den Test" {
		t.Errorf("Translate returned %q, want the command's answer without its trailing newline", translated)
	}
}

// What the command complained about is the only clue to why it failed, so it is
// what the popup says.
func TestACommandThatFailsIsReportedWithWhatItComplainedAbout(t *testing.T) {
	translator := command.New("echo 'model de-en not installed' >&2; exit 3")

	_, err := translator.Translate(context.Background(), "Bitte behebe den Test")

	if err == nil {
		t.Fatal("Translate returned no error for a command that failed")
	}
	if !strings.Contains(err.Error(), "model de-en not installed") {
		t.Errorf("Translate says %v, want what the command complained about", err)
	}
}

func TestACommandThatIsNotInstalledSaysSo(t *testing.T) {
	translator := command.New("herdr-polyglot-no-such-translator -m de-en")

	_, err := translator.Translate(context.Background(), "Bitte behebe den Test")

	if err == nil {
		t.Fatal("Translate returned no error for a command that is not there")
	}
	if !strings.Contains(err.Error(), "could not be started") {
		t.Errorf("Translate says %v, want it to say the command could not be started", err)
	}
}

// An empty answer would reach the agent as an empty prompt, which says nothing
// about a translator that is installed but did not translate.
func TestACommandThatAnswersWithNothingIsAFailure(t *testing.T) {
	translator := command.New("true")

	_, err := translator.Translate(context.Background(), "Bitte behebe den Test")

	if err == nil {
		t.Fatal("Translate returned no error for a command that answered with nothing")
	}
	if !strings.Contains(err.Error(), "nothing") {
		t.Errorf("Translate says %v, want it to say nothing came back", err)
	}
}

// A command that never answers must not hold the popup, and must not be left
// running behind it either.
func TestACommandThatHangsIsGivenUpOnAndKilled(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "still-running")
	translator := command.New(
		"sleep 30; touch "+marker, command.WithTimeout(100*time.Millisecond))

	started := time.Now()
	_, err := translator.Translate(context.Background(), "Bitte behebe den Test")

	if err == nil {
		t.Fatal("Translate returned no error for a command that never answered")
	}
	if !strings.Contains(err.Error(), "did not answer in time") {
		t.Errorf("Translate says %v, want it to say the command did not answer in time", err)
	}
	if waited := time.Since(started); waited > 5*time.Second {
		t.Errorf("Translate waited %v, want it to give up after the timeout", waited)
	}

	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Error("the command went on running after the popup gave up on it")
	}
}

// Live translation drops the request it no longer needs, and a dropped request is
// nothing to report: the popup must not show the last keystroke as a failure.
func TestADroppedRequestIsNotReportedAsAFailure(t *testing.T) {
	dropped, drop := context.WithCancel(context.Background())
	drop()

	_, err := command.New("cat").Translate(dropped, "Bitte behebe den Test")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Translate returned %v, want it to say the request was dropped", err)
	}
}

// The command usually carries the language itself, in a model name or a flag, but
// a script that wants it can read it.
func TestTheTargetLanguageIsPassedToTheCommand(t *testing.T) {
	translator := command.New(
		`printf '%s' "$POLYGLOT_TARGET_LANGUAGE"`, command.WithTargetLanguage("EN-GB"))

	translated, err := translator.Translate(context.Background(), "Bitte behebe den Test")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "EN-GB" {
		t.Errorf("the command read %q as the target language, want EN-GB", translated)
	}
}
