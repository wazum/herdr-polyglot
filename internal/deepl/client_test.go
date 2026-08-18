package deepl_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wazum/herdr-deepl-prompt/internal/deepl"
)

type capturedRequest struct {
	authorization string
	body          map[string]any
}

func serverReturning(t *testing.T, status int, response string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.authorization = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured.body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func TestTranslateReturnsTheEnglishTextForTheDraft(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK,
		`{"translations":[{"detected_source_language":"DE","text":"Please fix the failing test"}]}`)
	client := deepl.New("key-123", deepl.WithEndpoint(server.URL))

	translated, err := client.Translate(context.Background(), "Bitte behebe den fehlschlagenden Test")

	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if translated != "Please fix the failing test" {
		t.Errorf("Translate returned %q, want the translated text", translated)
	}
	if captured.authorization != "DeepL-Auth-Key key-123" {
		t.Errorf("request carried authorization %q, want the DeepL auth scheme", captured.authorization)
	}
	if got, ok := captured.body["text"].([]any); !ok || len(got) != 1 || got[0] != "Bitte behebe den fehlschlagenden Test" {
		t.Errorf("request sent text %v, want the draft as a single entry", captured.body["text"])
	}
	if captured.body["target_lang"] != "EN-US" {
		t.Errorf("request asked for %v, want EN-US", captured.body["target_lang"])
	}
}

func TestTranslateReportsAnUnsuccessfulResponse(t *testing.T) {
	t.Parallel()
	server, _ := serverReturning(t, http.StatusForbidden, `{"message":"Authorization failed"}`)
	client := deepl.New("wrong-key", deepl.WithEndpoint(server.URL))

	_, err := client.Translate(context.Background(), "Bitte behebe den Test")

	if err == nil {
		t.Fatal("Translate returned no error, want the rejected request")
	}
	if !strings.Contains(err.Error(), "Authorization failed") {
		t.Errorf("error %q does not carry DeepL's explanation", err)
	}
}

func TestTranslateReportsAResponseWithoutTranslations(t *testing.T) {
	t.Parallel()
	server, _ := serverReturning(t, http.StatusOK, `{"translations":[]}`)
	client := deepl.New("key-123", deepl.WithEndpoint(server.URL))

	_, err := client.Translate(context.Background(), "Bitte behebe den Test")

	if err == nil {
		t.Fatal("Translate returned no error, want a complaint about the empty response")
	}
}

func TestFreeApiKeysUseTheFreeEndpoint(t *testing.T) {
	t.Parallel()

	if endpoint := deepl.New("abc123:fx").Endpoint(); endpoint != "https://api-free.deepl.com/v2/translate" {
		t.Errorf("free key resolved to %q, want the api-free host", endpoint)
	}
	if endpoint := deepl.New("abc123").Endpoint(); endpoint != "https://api.deepl.com/v2/translate" {
		t.Errorf("pro key resolved to %q, want the api host", endpoint)
	}
}
