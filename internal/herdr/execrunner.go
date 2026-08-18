package herdr

import (
	"bytes"
	"context"
	"fmt"
	"os"
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
	command.Env = withoutCredentials(os.Environ())

	if err := command.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return fmt.Errorf("herdr %s: %w: %s", args[0], err, message)
		}
		return fmt.Errorf("herdr %s: %w", args[0], err)
	}
	return nil
}

// withoutCredentials keeps API keys out of the herdr process, which needs none
// of them and would pass them on to anything it starts.
func withoutCredentials(environment []string) []string {
	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if strings.Contains(name, "API_KEY") {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}
