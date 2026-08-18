package herdr

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ExecRunner struct{ binary string }

func NewExecRunner(binary string) ExecRunner {
	return ExecRunner{binary: binary}
}

func (e ExecRunner) Run(ctx context.Context, args ...string) error {
	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, e.binary, args...)
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return fmt.Errorf("herdr %s: %w: %s", args[0], err, message)
		}
		return fmt.Errorf("herdr %s: %w", args[0], err)
	}
	return nil
}
