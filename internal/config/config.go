// Package config resolves the plugin's settings from the environment herdr
// provides and from the .env file in the plugin's own config directory.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	targetVar    = "HERDR_DEEPL_TARGET"
	apiKeyVar    = "DEEPL_API_KEY"
	submitVar    = "HERDR_DEEPL_SUBMIT"
	dryRunVar    = "HERDR_DEEPL_DRY_RUN"
	languageVar  = "HERDR_DEEPL_LANGUAGE"
	configDirVar = "HERDR_PLUGIN_CONFIG_DIR"
	binaryVar    = "HERDR_BIN_PATH"

	defaultLanguage = "EN-US"
	defaultBinary   = "herdr"
	dotenvName      = ".env"
)

type Settings struct {
	Target         string
	APIKey         string
	TargetLanguage string
	HerdrBinary    string
	Submit         bool
	DryRun         bool
}

// Load merges herdr's injected environment with the plugin's .env file. The
// environment wins, so a one-off invocation can override stored settings.
func Load(getenv func(string) string) (Settings, error) {
	configDir := getenv(configDirVar)
	stored := readDotenv(filepath.Join(configDir, dotenvName))

	lookup := func(key string) string {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			return value
		}
		return stored[key]
	}

	settings := Settings{
		Target:         lookup(targetVar),
		APIKey:         lookup(apiKeyVar),
		TargetLanguage: orDefault(lookup(languageVar), defaultLanguage),
		HerdrBinary:    orDefault(lookup(binaryVar), defaultBinary),
		Submit:         !isDisabled(lookup(submitVar)),
		DryRun:         isEnabled(lookup(dryRunVar)),
	}

	if settings.Target == "" {
		return Settings{}, fmt.Errorf("no target pane: %s is not set", targetVar)
	}
	if settings.APIKey == "" && !settings.DryRun {
		return Settings{}, fmt.Errorf("no DeepL API key: set %s or add %s=... to %s",
			apiKeyVar, apiKeyVar, filepath.Join(configDir, dotenvName))
	}
	return settings, nil
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func isDisabled(value string) bool {
	switch strings.ToLower(value) {
	case "0", "false", "no", "off":
		return true
	default:
		return false
	}
}

func isEnabled(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
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
