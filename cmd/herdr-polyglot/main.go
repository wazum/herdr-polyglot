// Command herdr-polyglot opens an overlay for composing a prompt in your
// own language and delivers it to a herdr agent as English.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/config"
	"github.com/wazum/herdr-polyglot/internal/deepl"
	"github.com/wazum/herdr-polyglot/internal/herdr"
	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-polyglot:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	settings, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	ctx, stopListening := signal.NotifyContext(ctx, os.Interrupt)
	defer stopListening()

	flow := promptflow.New(translator(settings), target(settings))
	program := tea.NewProgram(
		overlay.New(ctx, flow),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("running overlay: %w", err)
	}
	return nil
}

func translator(settings config.Settings) promptflow.Translator {
	if settings.DryRun {
		return translation.DryRun{}
	}
	return deepl.New(settings.APIKey, deepl.WithTargetLanguage(settings.TargetLanguage))
}

func target(settings config.Settings) promptflow.Target {
	runner := herdr.NewExecRunner(settings.HerdrBinary)
	if settings.Submit {
		return herdr.NewAgentPrompt(runner, settings.Target)
	}
	return herdr.NewPaneText(runner, settings.Target)
}
