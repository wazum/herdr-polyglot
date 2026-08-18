package herdr_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/herdr"
)

func writeFakeBinary(t *testing.T, body string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "fake-herdr")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("writing fake binary: %v", err)
	}
	return binary
}

func TestExecRunnerPassesEveryArgumentToTheBinaryVerbatim(t *testing.T) {
	recorded := filepath.Join(t.TempDir(), "args")
	binary := writeFakeBinary(t, "printf '%s\\0' \"$@\" > "+recorded+"\n")

	err := herdr.NewExecRunner(binary).
		Run(context.Background(), "pane", "send-text", "w1:p3", "erste Zeile\nzweite Zeile")
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	raw, readErr := os.ReadFile(recorded)
	if readErr != nil {
		t.Fatalf("reading recorded arguments: %v", readErr)
	}
	got := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	want := []string{"pane", "send-text", "w1:p3", "erste Zeile\nzweite Zeile"}
	if !slices.Equal(got, want) {
		t.Errorf("binary received %q, want %q", got, want)
	}
}

func TestExecRunnerReportsTheBinaryFailureOutput(t *testing.T) {
	binary := writeFakeBinary(t, "echo 'pane w1:p3 not found' >&2\nexit 3\n")

	err := herdr.NewExecRunner(binary).Run(context.Background(), "pane", "send-text", "w1:p3", "hallo")

	if err == nil {
		t.Fatal("Run returned no error, want the binary failure")
	}
	if !strings.Contains(err.Error(), "pane w1:p3 not found") {
		t.Errorf("error %q does not mention the binary's stderr", err)
	}
}

func TestTheChildNeverSeesTheApiKey(t *testing.T) {
	recorded := filepath.Join(t.TempDir(), "env")
	binary := writeFakeBinary(t, "env > "+recorded+"\n")

	t.Setenv("HERDR_POLYGLOT_API_KEY", "secret-key-value")
	t.Setenv("DEEPL_API_KEY", "another-secret")
	t.Setenv("HERDR_POLYGLOT_TARGET", "w1:p3")

	if err := herdr.NewExecRunner(binary).Run(context.Background(), "pane", "list"); err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	raw, err := os.ReadFile(recorded)
	if err != nil {
		t.Fatalf("reading the child environment: %v", err)
	}
	environment := string(raw)
	for _, secret := range []string{"secret-key-value", "another-secret"} {
		if strings.Contains(environment, secret) {
			t.Errorf("the child environment carries %q; herdr needs no credentials", secret)
		}
	}
	if !strings.Contains(environment, "HERDR_POLYGLOT_TARGET=w1:p3") {
		t.Error("the child lost the rest of the environment, want only keys removed")
	}
}
