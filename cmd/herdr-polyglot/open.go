package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wazum/herdr-polyglot/internal/config"
	"github.com/wazum/herdr-polyglot/internal/herdr"
	"github.com/wazum/herdr-polyglot/internal/overlay"
)

const (
	pluginIDEnv = "HERDR_PLUGIN_ID"
	socketEnv   = "HERDR_SOCKET_PATH"
	paneEnv     = "HERDR_PANE_ID"
)

// runOpen puts the overlay in a floating popup over the pane the keybinding was
// pressed in, so the agent's output stays readable around it. Only the socket
// API can size a popup, which is why this does not go through the CLI.
func runOpen(ctx context.Context, args []string) error {
	pane := os.Getenv(paneEnv)
	if pane == "" {
		return fmt.Errorf("no invoking pane: %s is not set; open this from an agent pane", paneEnv)
	}

	submit := "1"
	for _, arg := range args {
		if arg == "--review" {
			submit = "0"
		}
	}

	live := loadedLive()
	socket := herdr.NewSocket(orDefault(os.Getenv(socketEnv), defaultSocketPath()))

	_, err := socket.OpenPopup(ctx, herdr.Popup{
		PluginID:   os.Getenv(pluginIDEnv),
		Entrypoint: "overlay",
		Width:      overlay.PopupWidth,
		Height:     overlay.PopupHeight(live),
		Env: map[string]string{
			"HERDR_POLYGLOT_TARGET": pane,
			"HERDR_POLYGLOT_SUBMIT": submit,
		},
	})
	return err
}

// loadedLive reads the stored settings to size the popup for the extra pane
// live mode shows. The invoking pane stands in for the target, which these
// settings do not need.
func loadedLive() bool {
	settings, err := config.Load(func(key string) string {
		if key == "HERDR_POLYGLOT_TARGET" {
			return os.Getenv(paneEnv)
		}
		return os.Getenv(key)
	})
	return err == nil && settings.Live
}

func defaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.config/herdr/herdr.sock"
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
