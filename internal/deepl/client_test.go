package deepl_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/deepl"
	"github.com/wazum/herdr-polyglot/internal/translation"
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

func TestTranslateWithContextSendsTheSurroundingsUnbilled(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK,
		`{"translations":[{"text":"Second sentence!"}]}`)
	client := deepl.New("key-123", deepl.WithEndpoint(server.URL))

	translated, err := client.TranslateWithContext(
		context.Background(), "Zweiter Satz!", "Erster Satz.")
	if err != nil {
		t.Fatalf("TranslateWithContext returned unexpected error: %v", err)
	}
	if translated != "Second sentence!" {
		t.Errorf("returned %q, want the translated sentence", translated)
	}
	if got, ok := captured.body["text"].([]any); !ok || len(got) != 1 || got[0] != "Zweiter Satz!" {
		t.Errorf("request translated %v, want only the sentence", captured.body["text"])
	}
	if captured.body["context"] != "Erster Satz." {
		t.Errorf("request carried context %v, want the surrounding draft", captured.body["context"])
	}
}

func TestTranslateSendsNoContextFieldWhenThereIsNone(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK,
		`{"translations":[{"text":"Please fix it"}]}`)
	client := deepl.New("key-123", deepl.WithEndpoint(server.URL))

	if _, err := client.Translate(context.Background(), "Bitte behebe es"); err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}

	if _, present := captured.body["context"]; present {
		t.Errorf("request carried a context field %v, want it omitted", captured.body["context"])
	}
}

func TestAPlainHttpEndpointIsRefusedSoTheKeyIsNotSentInClear(t *testing.T) {
	t.Parallel()

	_, err := deepl.Provider{}.New(translation.Options{
		APIKey:   "key-123",
		Endpoint: "http://translate.example/v2",
	})

	if err == nil {
		t.Fatal("New returned no error, want the plain-text endpoint refused")
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("error %q does not explain that https is required", err)
	}
}

func TestALocalHttpEndpointIsAllowedForTesting(t *testing.T) {
	t.Parallel()

	if _, err := (deepl.Provider{}).New(translation.Options{
		APIKey:   "key-123",
		Endpoint: "http://127.0.0.1:8080/v2",
	}); err != nil {
		t.Errorf("New refused a loopback endpoint: %v", err)
	}
}

func TestAnEnormousResponseIsNotReadWithoutLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translations":[{"text":"`))
		for range 4096 {
			if _, err := w.Write(bytes.Repeat([]byte("A"), 1024)); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)

	_, err := deepl.New("key-123", deepl.WithEndpoint(server.URL)).
		Translate(context.Background(), "Bitte behebe es")

	if err == nil {
		t.Error("Translate accepted an unbounded response, want it cut off")
	}
}
