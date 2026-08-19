package command_test

import (
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/command"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

func TestTheProviderIsNamedCmd(t *testing.T) {
	if name := (command.Provider{}).Name(); name != "cmd" {
		t.Errorf("Name() = %q, want cmd", name)
	}
}

func TestAProviderWithNoCommandSaysWhichSettingIsMissing(t *testing.T) {
	_, err := command.Provider{}.New(translation.Options{})

	if err == nil {
		t.Fatal("New returned no error with nothing to run")
	}
	if !strings.Contains(err.Error(), "HERDR_POLYGLOT_COMMAND") {
		t.Errorf("New says %v, want it to name the setting that is missing", err)
	}
}
