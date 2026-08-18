package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wazum/herdr-deepl-prompt/internal/config"
)

func envFrom(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func configDirContaining(t *testing.T, dotenv string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600); err != nil {
		t.Fatalf("writing .env: %v", err)
	}
	return dir
}

func TestLoadTakesTargetAndApiKeyFromTheEnvironment(t *testing.T) {
	t.Parallel()

	settings, err := config.Load(envFrom(map[string]string{
		"HERDR_DEEPL_TARGET": "w1:p3",
		"DEEPL_API_KEY":      "key-123",
	}))

	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if settings.Target != "w1:p3" {
		t.Errorf("Target is %q, want w1:p3", settings.Target)
	}
	if settings.APIKey != "key-123" {
		t.Errorf("APIKey is %q, want key-123", settings.APIKey)
	}
	if !settings.Submit {
		t.Error("Submit is false, want submitting by default")
	}
	if settings.TargetLanguage != "EN-US" {
		t.Errorf("TargetLanguage is %q, want EN-US by default", settings.TargetLanguage)
	}
	if settings.HerdrBinary != "herdr" {
		t.Errorf("HerdrBinary is %q, want herdr by default", settings.HerdrBinary)
	}
}

func TestLoadHonoursOverridesForSubmittingAndLanguage(t *testing.T) {
	t.Parallel()

	settings, err := config.Load(envFrom(map[string]string{
		"HERDR_DEEPL_TARGET":   "w1:p3",
		"DEEPL_API_KEY":        "key-123",
		"HERDR_DEEPL_SUBMIT":   "0",
		"HERDR_DEEPL_LANGUAGE": "EN-GB",
		"HERDR_BIN_PATH":       "/opt/homebrew/bin/herdr",
	}))

	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if settings.Submit {
		t.Error("Submit is true, want it disabled by HERDR_DEEPL_SUBMIT=0")
	}
	if settings.TargetLanguage != "EN-GB" {
		t.Errorf("TargetLanguage is %q, want EN-GB", settings.TargetLanguage)
	}
	if settings.HerdrBinary != "/opt/homebrew/bin/herdr" {
		t.Errorf("HerdrBinary is %q, want the path from HERDR_BIN_PATH", settings.HerdrBinary)
	}
}

func TestLoadReadsSettingsFromTheDotEnvInThePluginConfigDirectory(t *testing.T) {
	t.Parallel()
	configDir := configDirContaining(t, "# DeepL credentials\nDEEPL_API_KEY=key-from-file\nHERDR_DEEPL_LANGUAGE=\"EN-GB\"\n")

	settings, err := config.Load(envFrom(map[string]string{
		"HERDR_DEEPL_TARGET":      "w1:p3",
		"HERDR_PLUGIN_CONFIG_DIR": configDir,
	}))

	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if settings.APIKey != "key-from-file" {
		t.Errorf("APIKey is %q, want the value from the .env file", settings.APIKey)
	}
	if settings.TargetLanguage != "EN-GB" {
		t.Errorf("TargetLanguage is %q, want the unquoted value from the .env file", settings.TargetLanguage)
	}
}

func TestTheEnvironmentWinsOverTheDotEnvFile(t *testing.T) {
	t.Parallel()
	configDir := configDirContaining(t, "DEEPL_API_KEY=key-from-file\n")

	settings, err := config.Load(envFrom(map[string]string{
		"HERDR_DEEPL_TARGET":      "w1:p3",
		"HERDR_PLUGIN_CONFIG_DIR": configDir,
		"DEEPL_API_KEY":           "key-from-environment",
	}))

	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if settings.APIKey != "key-from-environment" {
		t.Errorf("APIKey is %q, want the environment to win", settings.APIKey)
	}
}

func TestLoadWithoutAnApiKeySaysWhereToPutOne(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	_, err := config.Load(envFrom(map[string]string{
		"HERDR_DEEPL_TARGET":      "w1:p3",
		"HERDR_PLUGIN_CONFIG_DIR": configDir,
	}))

	if err == nil {
		t.Fatal("Load returned no error, want a complaint about the missing key")
	}
	if !strings.Contains(err.Error(), "DEEPL_API_KEY") || !strings.Contains(err.Error(), configDir) {
		t.Errorf("error %q names neither the variable nor the config directory", err)
	}
}

func TestLoadWithoutAnApiKeyIsFineForADryRun(t *testing.T) {
	t.Parallel()

	settings, err := config.Load(envFrom(map[string]string{
		"HERDR_DEEPL_TARGET":  "w1:p3",
		"HERDR_DEEPL_DRY_RUN": "1",
	}))

	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if !settings.DryRun {
		t.Error("DryRun is false, want it enabled")
	}
}

func TestLoadWithoutATargetFails(t *testing.T) {
	t.Parallel()

	_, err := config.Load(envFrom(map[string]string{"DEEPL_API_KEY": "key-123"}))

	if err == nil {
		t.Fatal("Load returned no error, want a complaint about the missing target")
	}
	if !strings.Contains(err.Error(), "HERDR_DEEPL_TARGET") {
		t.Errorf("error %q does not name the target variable", err)
	}
}
