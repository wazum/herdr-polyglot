// Command herdr-polyglot opens an overlay for composing a prompt in your own
// language and delivers it to a herdr agent in the target language.
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
	if len(os.Args) > 1 && os.Args[1] == "open" {
		if err := runOpen(context.Background(), os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "herdr-polyglot:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-polyglot:", err)
		os.Exit(1)
	}
}

// services lists the translation services this build offers. The first one is
// the default; adding another service means adding it here.
func services() *translation.Registry {
	return translation.NewRegistry(
		deepl.Provider{},
		translation.DryRunProvider{},
	)
}

func run(ctx context.Context) error {
	settings, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	registry := services()
	service := settings.Provider
	if service == "" {
		service = registry.Default()
	}

	translator, err := registry.Translator(service, settings.Options)
	if err != nil {
		return fmt.Errorf("%w; configure it in %s", err, settings.ConfigFile)
	}
	if settings.Live {
		// Writing means translating the same draft again and again, so pay for
		// each sentence once instead of for the whole draft every time.
		translator = translation.Segmented(translator)
	}

	ctx, stopListening := signal.NotifyContext(ctx, os.Interrupt)
	defer stopListening()

	flow := promptflow.New(translator, target(settings.HerdrBinary, settings.Target, settings.Submit))
	program := tea.NewProgram(
		overlay.New(ctx, flow, overlay.Options{
			Service:  service,
			Language: settings.Options.TargetLanguage,
			Review:   !settings.Submit,
			Vim:      settings.Vim,
			Live:     settings.Live,
		}),
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("running overlay: %w", err)
	}
	return nil
}

func target(binary, pane string, submit bool) promptflow.Target {
	runner := herdr.NewExecRunner(binary)
	if submit {
		return herdr.NewAgentPrompt(runner, pane)
	}
	return herdr.NewPaneText(runner, pane)
}
