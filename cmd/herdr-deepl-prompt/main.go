// Command herdr-deepl-prompt opens an overlay for composing a prompt in your
// own language and delivers it to a herdr agent as English.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wazum/herdr-deepl-prompt/internal/herdr"
	"github.com/wazum/herdr-deepl-prompt/internal/overlay"
	"github.com/wazum/herdr-deepl-prompt/internal/promptflow"
	"github.com/wazum/herdr-deepl-prompt/internal/translation"
)

const (
	targetEnv = "HERDR_DEEPL_TARGET"
	binaryEnv = "HERDR_BIN_PATH"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-deepl-prompt:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("herdr-deepl-prompt", flag.ContinueOnError)
	target := flags.String("target", os.Getenv(targetEnv), "pane or agent receiving the prompt")
	submit := flags.Bool("submit", true, "submit the prompt instead of only typing it into the input")
	dryRun := flags.Bool("dry-run", false, "mark the draft instead of translating it")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *target == "" {
		return fmt.Errorf("no target: pass --target or set %s", targetEnv)
	}
	if !*dryRun {
		return errors.New("only --dry-run is wired up so far")
	}

	ctx, stopListening := signal.NotifyContext(ctx, os.Interrupt)
	defer stopListening()

	flow := promptflow.New(translation.DryRun{}, deliveryTarget(*target, *submit))
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

func deliveryTarget(target string, submit bool) promptflow.Target {
	runner := herdr.NewExecRunner(herdrBinary())
	if submit {
		return herdr.NewAgentPrompt(runner, target)
	}
	return herdr.NewPaneText(runner, target)
}

func herdrBinary() string {
	if binary := os.Getenv(binaryEnv); binary != "" {
		return binary
	}
	return "herdr"
}
