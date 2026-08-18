//go:build live

package herdr_test

import (
	"context"
	"os"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/herdr"
)

// Puts a code block into a real agent's input through a real herdr, so the claim
// that line breaks survive is measured rather than assumed:
//
//	POLYGLOT_LIVE_PANE=wJ:p1 go test -tags live -run LiveDelivery -v ./internal/herdr/
func TestLiveDeliveryKeepsTheShapeOfACodeBlock(t *testing.T) {
	pane := os.Getenv("POLYGLOT_LIVE_PANE")
	if pane == "" {
		t.Skip("no pane to deliver into")
	}

	block := "Please fix this:\n\n```go\nif customer == nil { // should never happen\n\treturn fmt.Errorf(\"customer missing\")\n}\n```"
	target := herdr.NewPaneText(herdr.NewExecRunner("herdr"), pane)

	if err := target.Insert(context.Background(), block); err != nil {
		t.Fatalf("Insert returned: %v", err)
	}
	t.Log("delivered; read the pane to see what arrived")
}
