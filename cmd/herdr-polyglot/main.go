// Command herdr-polyglot opens an overlay for composing a prompt in your own
// language and delivers it to a herdr agent in the target language.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-polyglot/internal/config"
	"github.com/wazum/herdr-polyglot/internal/deepl"
	"github.com/wazum/herdr-polyglot/internal/draft"
	"github.com/wazum/herdr-polyglot/internal/herdr"
	"github.com/wazum/herdr-polyglot/internal/overlay"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "open":
			if err := runOpen(context.Background(), os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "herdr-polyglot:", err)
				os.Exit(1)
			}
			return

		// warm does nothing but exist: running it puts the binary in the page
		// cache, so the keypress that opens the popup does not pay for that.
		case "warm":
			return
		}
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
		if settings.ConfigFile == "" {
			return err
		}
		return fmt.Errorf("%w; configure it in %s", err, settings.ConfigFile)
	}

	// Herdr closes a popup by hanging up on it. Ending on the signal instead of
	// dying on it is what lets the draft be kept below.
	ctx, stopListening := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stopListening()

	runner := herdr.NewExecRunner(settings.HerdrBinary)
	flow := promptflow.New(translator,
		herdr.NewAgentPrompt(runner, settings.Target),
		herdr.NewPaneText(runner, settings.Target),
		// Writing means translating the same draft again and again, so a preview
		// pays for each sentence once. Live translation can be switched on at any
		// time, so this is set up whether it starts on or off.
		promptflow.WithPreviewTranslator(translation.Segmented(translator)),
	)
	program := tea.NewProgram(
		overlay.New(ctx, flow, overlay.Options{
			Service:  service,
			Language: settings.Options.TargetLanguage,
			Review:   !settings.Submit,
			Vim:      settings.Vim,
			Live:     settings.Live,
			Confirm:  settings.Confirm,
			Pulse:    settings.Pulse,
			MaxDraft: settings.MaxDraft,

			Drafts: drafts(settings.KeepDraft, settings.StateDir, settings.Target),
		}),
		tea.WithContext(ctx),
		// No alternate screen: entering it clears the pane to the terminal's own
		// background, which then differs from the popup herdr painted around it.
		// Drawing inline leaves every cell we do not write as herdr left it.
	)
	final, err := program.Run()
	if kept, ok := final.(overlay.Model); ok {
		if err := kept.KeepUnfinished(); err != nil {
			fmt.Fprintln(os.Stderr, "polyglot:", err)
		}
	}
	// A closed popup ends the program this way; it is not a failure.
	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return fmt.Errorf("running overlay: %w", err)
	}
	return nil
}

// drafts keeps an unfinished prompt for this pane, unless the author would
// rather start from an empty box every time.
func drafts(keep bool, stateDir, target string) overlay.Drafts {
	if !keep || stateDir == "" {
		return nil
	}
	return draft.NewStore(stateDir).For(target)
}
