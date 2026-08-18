//go:build live

package promptflow_test

import (
	"context"
	"os"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/deepl"
	"github.com/wazum/herdr-polyglot/internal/herdr"
	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

// The whole way through, into a real agent's input: translated, code protected,
// delivered with its lines intact.
//
//	HERDR_PLUGIN_CONFIG_DIR="$(herdr plugin config-dir wazum.polyglot)" \
//	POLYGLOT_LIVE_PANE=wJ:p1 go test -tags live -run LiveWholeWay -v ./internal/promptflow/
func TestLiveWholeWayIntoAnAgentsInput(t *testing.T) {
	pane := os.Getenv("POLYGLOT_LIVE_PANE")
	key := os.Getenv("HERDR_POLYGLOT_API_KEY")
	if pane == "" || key == "" {
		t.Skip("needs a pane and a key")
	}

	service, err := deepl.Provider{}.New(translation.Options{APIKey: key, TargetLanguage: "EN-US"})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	target := herdr.NewPaneText(herdr.NewExecRunner("herdr"), pane)
	flow := promptflow.New(translation.Protecting(service), target, target)

	draft := "Bitte behebe den Test in `berechneRabatt()`. Hier der Fehler:\n\n" +
		"```go\nif kunde == nil { // sollte nie passieren\n\treturn fmt.Errorf(\"kunde fehlt\")\n}\n```"

	english, err := flow.Submit(context.Background(), draft, promptflow.Typing)
	if err != nil {
		t.Fatalf("Submit returned: %v", err)
	}
	t.Logf("delivered:\n%s", english)
}
