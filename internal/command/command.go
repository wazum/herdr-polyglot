// Package command translates through a program on the machine instead of a
// service on the network. It is one service behind the translation ports, and
// knows nothing about the overlay.
package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	// A local model can take its time loading before it translates a word, and a
	// command that hangs must still not hold the popup for good.
	defaultTimeout = 60 * time.Second

	// Long enough for a program that is shutting down to finish writing, short
	// enough that one which ignores the signal does not hold the popup.
	waitDelay = time.Second

	maxComplaintLength = 200

	// The command usually carries the language itself; this is for a script that
	// would rather read it.
	languageVar = "POLYGLOT_TARGET_LANGUAGE"
)

type Translator struct {
	commandLine    string
	targetLanguage string
	timeout        time.Duration
}

type Option func(*Translator)

func WithTargetLanguage(language string) Option {
	return func(t *Translator) { t.targetLanguage = language }
}

func WithTimeout(timeout time.Duration) Option {
	return func(t *Translator) { t.timeout = timeout }
}

func New(commandLine string, options ...Option) *Translator {
	translator := &Translator{commandLine: commandLine, timeout: defaultTimeout}
	for _, apply := range options {
		apply(translator)
	}
	return translator
}

func (t *Translator) Translate(ctx context.Context, draft string) (string, error) {
	ctx, done := context.WithTimeout(ctx, t.timeout)
	defer done()

	// A shell so a command line can be a pipeline with quoting in it, as it is in
	// the config file it was written in. The draft is only ever written to the
	// program's input, so nothing typed in the popup can become shell syntax.
	running := exec.CommandContext(ctx, "sh", "-c", t.commandLine) // #nosec G204
	running.Stdin = strings.NewReader(draft)
	running.Env = append(os.Environ(), languageVar+"="+t.targetLanguage)

	// The command line may start several programs, and killing the shell alone
	// leaves them running and holding the output open. They are given a process
	// group of their own so the whole pipeline goes at once.
	running.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	running.Cancel = func() error {
		return syscall.Kill(-running.Process.Pid, syscall.SIGKILL)
	}
	running.WaitDelay = waitDelay

	var answer, complaint bytes.Buffer
	running.Stdout = &answer
	running.Stderr = &complaint

	if err := running.Run(); err != nil {
		return "", t.trouble(ctx, err, complaint.String())
	}
	if answer.Len() == 0 {
		return "", fmt.Errorf("%s returned nothing", t.program())
	}
	return strings.TrimRight(answer.String(), "\r\n"), nil
}

// A program that is not installed, one that refused the draft and one that hung
// are three different things to do something about, so they read differently.
func (t *Translator) trouble(ctx context.Context, err error, complaint string) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return ctx.Err()
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("%s did not answer in time", t.program())
	}

	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		return fmt.Errorf("%s could not be run: %w", t.program(), err)
	}

	// The shell reports a command it cannot find or cannot execute with these two,
	// which is a setting to correct rather than a translation that went wrong.
	if code := exit.ExitCode(); code == 126 || code == 127 {
		return fmt.Errorf("%s could not be started — is it installed?", t.program())
	}
	if said := firstLine(complaint); said != "" {
		return fmt.Errorf("%s failed: %s", t.program(), said)
	}
	return fmt.Errorf("%s failed with exit status %d", t.program(), exit.ExitCode())
}

// The command line can be a pipeline with paths and flags in it; what is worth
// naming in a popup is the program it starts with.
func (t *Translator) program() string {
	first, _, _ := strings.Cut(strings.TrimSpace(t.commandLine), " ")
	if first == "" {
		return "the command"
	}
	return filepath.Base(first)
}

func firstLine(complaint string) string {
	for line := range strings.SplitSeq(complaint, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return cutTo(line, maxComplaintLength)
		}
	}
	return ""
}

func cutTo(line string, length int) string {
	letters := []rune(line)
	if len(letters) <= length {
		return line
	}
	return string(letters[:length]) + " …"
}
