package herdr

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
)

// Socket talks to herdr's API directly. The CLI covers most of it, but sized
// floating popups are only reachable over the socket.
type Socket struct{ path string }

func NewSocket(path string) Socket {
	return Socket{path: path}
}

type Popup struct {
	PluginID   string
	Entrypoint string
	// Width and Height are cells, or a string such as "70%".
	Width  any
	Height any
	Env    map[string]string
}

func (s Socket) OpenPopup(ctx context.Context, popup Popup) (string, error) {
	params := map[string]any{
		"plugin_id":  popup.PluginID,
		"entrypoint": popup.Entrypoint,
		"placement":  "popup",
		"focus":      true,
	}
	if popup.Width != nil {
		params["width"] = popup.Width
	}
	if popup.Height != nil {
		params["height"] = popup.Height
	}
	if len(popup.Env) > 0 {
		params["env"] = popup.Env
	}

	raw, err := s.call(ctx, "plugin.pane.open", params)
	if err != nil {
		return "", err
	}

	var opened struct {
		PluginPane struct {
			Pane struct {
				PaneID string `json:"pane_id"`
			} `json:"pane"`
		} `json:"plugin_pane"`
	}
	if err := json.Unmarshal(raw, &opened); err != nil {
		return "", fmt.Errorf("reading opened pane: %w", err)
	}
	return opened.PluginPane.Pane.PaneID, nil
}

func (s Socket) call(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	connection, err := (&net.Dialer{}).DialContext(ctx, "unix", s.path)
	if err != nil {
		return nil, fmt.Errorf("connecting to herdr: %w", err)
	}
	defer connection.Close()

	request, err := json.Marshal(map[string]any{
		"id":     "polyglot",
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding %s: %w", method, err)
	}
	if _, err := connection.Write(append(request, '\n')); err != nil {
		return nil, fmt.Errorf("sending %s: %w", method, err)
	}

	line, err := bufio.NewReader(connection).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return nil, fmt.Errorf("reading the reply to %s: %w", method, err)
	}

	var reply struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(line, &reply); err != nil {
		return nil, fmt.Errorf("reading the reply to %s: %w", method, err)
	}
	if reply.Error != nil {
		return nil, fmt.Errorf("%s: %s: %s", method, reply.Error.Code, reply.Error.Message)
	}
	return reply.Result, nil
}
