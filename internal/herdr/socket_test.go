package herdr_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/herdr"
)

type fakeServer struct {
	path     string
	reply    string
	mu       sync.Mutex
	received []map[string]any
}

func listen(t *testing.T, reply string) *fakeServer {
	t.Helper()
	// A socket path is limited to about a hundred characters, which the long
	// per-test TempDir path exceeds.
	directory, err := os.MkdirTemp("/tmp", "pg")
	if err != nil {
		t.Fatalf("creating socket directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })

	server := &fakeServer{path: filepath.Join(directory, "s"), reply: reply}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", server.path)
	if err != nil {
		t.Fatalf("listening on %s: %v", server.path, err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go server.serve(connection)
		}
	}()
	return server
}

func (f *fakeServer) serve(connection net.Conn) {
	defer connection.Close()

	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		return
	}

	var request map[string]any
	if err := json.Unmarshal([]byte(line), &request); err == nil {
		f.mu.Lock()
		f.received = append(f.received, request)
		f.mu.Unlock()
	}
	_, _ = connection.Write([]byte(f.reply + "\n"))
}

func (f *fakeServer) lastRequest(t *testing.T) map[string]any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.received) == 0 {
		t.Fatal("the server received no request")
	}
	return f.received[len(f.received)-1]
}

func TestOpeningAPopupAsksForAFloatingPaneOverTheAgent(t *testing.T) {
	t.Parallel()
	server := listen(t, `{"id":"polyglot","result":{"plugin_pane":{"pane":{"pane_id":"w1:p9"}}}}`)

	pane, err := herdr.NewSocket(server.path).OpenPopup(context.Background(), herdr.Popup{
		PluginID:   "wazum.polyglot",
		Entrypoint: "overlay",
		Width:      "70%",
		Height:     14,
		Env:        map[string]string{"HERDR_POLYGLOT_TARGET": "w1:p3"},
	})
	if err != nil {
		t.Fatalf("OpenPopup returned unexpected error: %v", err)
	}
	if pane != "w1:p9" {
		t.Errorf("OpenPopup returned pane %q, want w1:p9", pane)
	}

	request := server.lastRequest(t)
	if request["method"] != "plugin.pane.open" {
		t.Errorf("called %v, want plugin.pane.open", request["method"])
	}

	params, ok := request["params"].(map[string]any)
	if !ok {
		t.Fatalf("params are %T, want an object", request["params"])
	}
	if params["placement"] != "popup" {
		t.Errorf("placement is %v, want popup so the agent stays visible", params["placement"])
	}
	if params["width"] != "70%" {
		t.Errorf("width is %v, want 70%%", params["width"])
	}
	if params["height"] != float64(14) {
		t.Errorf("height is %v, want 14 cells", params["height"])
	}
	if params["focus"] != true {
		t.Errorf("focus is %v, want the popup focused", params["focus"])
	}

	env, ok := params["env"].(map[string]any)
	if !ok || env["HERDR_POLYGLOT_TARGET"] != "w1:p3" {
		t.Errorf("env is %v, want the target pane passed on", params["env"])
	}
}

func TestOpeningAPopupReportsWhatHerdrRefused(t *testing.T) {
	t.Parallel()
	server := listen(t, `{"id":"polyglot","error":{"code":"invalid_params","message":"popup too large"}}`)

	_, err := herdr.NewSocket(server.path).OpenPopup(context.Background(), herdr.Popup{
		PluginID:   "wazum.polyglot",
		Entrypoint: "overlay",
	})

	if err == nil {
		t.Fatal("OpenPopup returned no error, want the refusal")
	}
	if !strings.Contains(err.Error(), "popup too large") {
		t.Errorf("error %q does not carry herdr's explanation", err)
	}
}

func TestOpeningAPopupWithoutASocketFails(t *testing.T) {
	t.Parallel()

	_, err := herdr.NewSocket(filepath.Join(t.TempDir(), "missing")).
		OpenPopup(context.Background(), herdr.Popup{PluginID: "p", Entrypoint: "overlay"})

	if err == nil {
		t.Fatal("OpenPopup returned no error, want a complaint about the socket")
	}
}
