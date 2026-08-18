package herdr_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/herdr"
)

type recordingRunner struct{ calls [][]string }

func (r *recordingRunner) Run(_ context.Context, args ...string) error {
	r.calls = append(r.calls, args)
	return nil
}

func TestAgentPromptSubmitsTheTextToTheAgent(t *testing.T) {
	runner := &recordingRunner{}

	err := herdr.NewAgentPrompt(runner, "w1:p3").Insert(context.Background(), "Please fix the failing test")
	if err != nil {
		t.Fatalf("Insert returned unexpected error: %v", err)
	}
	want := []string{"agent", "prompt", "w1:p3", "Please fix the failing test"}
	if len(runner.calls) != 1 || !slices.Equal(runner.calls[0], want) {
		t.Errorf("runner saw %v, want a single call %v", runner.calls, want)
	}
}

func TestPaneTextInsertsTheTextWithoutSubmitting(t *testing.T) {
	runner := &recordingRunner{}

	err := herdr.NewPaneText(runner, "w1:p3").Insert(context.Background(), "Please fix the failing test")
	if err != nil {
		t.Fatalf("Insert returned unexpected error: %v", err)
	}
	want := []string{"pane", "send-text", "w1:p3", "Please fix the failing test"}
	if len(runner.calls) != 1 || !slices.Equal(runner.calls[0], want) {
		t.Errorf("runner saw %v, want a single call %v", runner.calls, want)
	}
}

func TestInsertWithoutATargetRunsNothing(t *testing.T) {
	runner := &recordingRunner{}

	err := herdr.NewAgentPrompt(runner, "").Insert(context.Background(), "Please fix the failing test")

	if !errors.Is(err, herdr.ErrNoTarget) {
		t.Errorf("Insert returned error %v, want ErrNoTarget", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("runner saw %v, want no calls", runner.calls)
	}
}
