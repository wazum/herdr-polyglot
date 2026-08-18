// Package config resolves the plugin's settings from the environment herdr
// provides and from the .env file in the plugin's own config directory. It
// stays neutral about which translation service is used.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

const (
	targetVar    = "HERDR_POLYGLOT_TARGET"
	providerVar  = "HERDR_POLYGLOT_PROVIDER"
	apiKeyVar    = "HERDR_POLYGLOT_API_KEY"
	languageVar  = "HERDR_POLYGLOT_LANGUAGE"
	endpointVar  = "HERDR_POLYGLOT_ENDPOINT"
	submitVar    = "HERDR_POLYGLOT_SUBMIT"
	vimVar       = "HERDR_POLYGLOT_VIM"
	liveVar      = "HERDR_POLYGLOT_LIVE"
	keepDraftVar = "HERDR_POLYGLOT_KEEP_DRAFT"
	confirmVar   = "HERDR_POLYGLOT_CONFIRM"
	maxDraftVar  = "HERDR_POLYGLOT_MAX_DRAFT"
	stateDirVar  = "HERDR_PLUGIN_STATE_DIR"
	configDirVar = "HERDR_PLUGIN_CONFIG_DIR"
	binaryVar    = "HERDR_BIN_PATH"

	defaultLanguage = "EN-US"
	defaultBinary   = "herdr"
	dotenvName      = ".env"
)

type Settings struct {
	Target string
	// Provider names the translation service; empty means the default.
	Provider   string
	Options    translation.Options
	ConfigFile string
	// StateDir is where an unfinished prompt is kept between sessions.
	StateDir    string
	HerdrBinary string
	Submit      bool
	Vim         bool
	Live        bool
	KeepDraft   bool
	Confirm     bool
	MaxDraft    int
}

// The environment wins over the .env file, so a one-off invocation can
// override stored settings. Credentials are passed along unchecked: only the
// service knows what it needs.
func Load(getenv func(string) string) (Settings, error) {
	// Without the directory herdr sets aside there is nothing of ours to read.
	// Falling back to a relative path would let a .env in whatever directory
	// the process started in decide the settings.
	configFile := ""
	stored := map[string]string{}
	if configDir := getenv(configDirVar); configDir != "" {
		configFile = filepath.Join(configDir, dotenvName)
		stored = readDotenv(configFile)
	}

	lookup := func(key string) string {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			return value
		}
		return stored[key]
	}

	provider := lookup(providerVar)
	settings := Settings{
		Target:      lookup(targetVar),
		Provider:    provider,
		ConfigFile:  configFile,
		StateDir:    getenv(stateDirVar),
		HerdrBinary: orDefault(lookup(binaryVar), defaultBinary),
		Submit:      !isDisabled(lookup(submitVar)),
		Vim:         isEnabled(lookup(vimVar)),
		Live:        isEnabled(lookup(liveVar)),
		KeepDraft:   !isDisabled(lookup(keepDraftVar)),
		Confirm:     isEnabled(lookup(confirmVar)),
		MaxDraft:    wholeNumber(lookup(maxDraftVar)),
		Options: translation.Options{
			APIKey:         orDefault(lookup(scopedKeyVar(provider)), lookup(apiKeyVar)),
			TargetLanguage: orDefault(lookup(languageVar), defaultLanguage),
			Endpoint:       lookup(endpointVar),
		},
	}

	if settings.Target == "" {
		return Settings{}, fmt.Errorf("no target pane: %s is not set", targetVar)
	}
	return settings, nil
}

// Scoping the key by service lets several services be configured side by side.
func scopedKeyVar(provider string) string {
	if provider == "" {
		return ""
	}
	scope := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(provider))
	return "HERDR_POLYGLOT_" + scope + "_API_KEY"
}

func wholeNumber(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 0 {
		return 0
	}
	return number
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func isEnabled(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func isDisabled(value string) bool {
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

// readDotenv parses KEY=VALUE lines, ignoring comments and blanks. A missing
// file is normal: the plugin works from the environment alone.
func readDotenv(path string) map[string]string {
	values := map[string]string{}

	file, err := os.Open(path)
	if err != nil {
		return values
	}
	defer file.Close()

	lines := bufio.NewScanner(file)
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}
