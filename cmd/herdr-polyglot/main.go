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
	"github.com/wazum/herdr-polyglot/internal/google"
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
		google.Provider{},
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
	flowOptions := []promptflow.Option{
		// Writing means translating the same draft again and again, so a preview
		// pays for each sentence once. Live translation can be switched on at any
		// time, so this is set up whether it starts on or off. Protecting sits
		// outside the cache, so a fenced block is taken out before the draft is
		// split into sentences.
		promptflow.WithPreviewTranslator(
			translation.Protecting(translation.Segmented(translator))),
	}
	if spending, err := spendingOf(translator); err == nil {
		flowOptions = append(flowOptions, promptflow.WithUsageReporter(spending))
	}

	flow := promptflow.New(translation.Protecting(translator),
		herdr.NewAgentPrompt(runner, settings.Target),
		herdr.NewPaneText(runner, settings.Target),
		flowOptions...,
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
			Logo:     settings.Logo,
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

// spendingOf asks the service what it has spent, in the application's words. Only
// this layer knows both vocabularies, which is what keeps the translation package
// out of the use case.
func spendingOf(translator translation.Translator) (promptflow.UsageReporter, error) {
	reporter, err := translation.ReporterOf(translator)
	if err != nil {
		return nil, err
	}
	return spendingReporter{reporter}, nil
}

type spendingReporter struct{ reporter translation.UsageReporter }

func (s spendingReporter) Usage(ctx context.Context) (promptflow.Usage, error) {
	spent, err := s.reporter.Usage(ctx)
	if err != nil {
		return promptflow.Usage{}, err
	}
	return promptflow.Usage{Used: spent.Used, Limit: spent.Limit}, nil
}

// drafts keeps an unfinished prompt for this pane, unless the author would
// rather start from an empty box every time.
func drafts(keep bool, stateDir, target string) overlay.Drafts {
	if !keep || stateDir == "" {
		return nil
	}
	return draft.NewStore(stateDir).For(target)
}
