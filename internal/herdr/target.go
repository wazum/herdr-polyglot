// Package herdr delivers a finished prompt into a herdr-managed pane.
package herdr

import (
	"context"
	"errors"
)

var ErrNoTarget = errors.New("no target pane or agent")

type Runner interface {
	Run(ctx context.Context, args ...string) error
}

type AgentPrompt struct {
	runner Runner
	target string
}

func NewAgentPrompt(runner Runner, target string) AgentPrompt {
	return AgentPrompt{runner: runner, target: target}
}

func (a AgentPrompt) Insert(ctx context.Context, text string) error {
	if a.target == "" {
		return ErrNoTarget
	}
	return a.runner.Run(ctx, "agent", "prompt", a.target, text)
}

// PaneText types the prompt into a pane and leaves submitting to the user.
type PaneText struct {
	runner Runner
	target string
}

func NewPaneText(runner Runner, target string) PaneText {
	return PaneText{runner: runner, target: target}
}

func (p PaneText) Insert(ctx context.Context, text string) error {
	if p.target == "" {
		return ErrNoTarget
	}
	return p.runner.Run(ctx, "pane", "send-text", p.target, text)
}
