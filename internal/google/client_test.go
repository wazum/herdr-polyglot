package google_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wazum/herdr-polyglot/internal/google"
)

type capturedRequest struct {
	key   string
	query string
	body  map[string]any
}

func serverReturning(t *testing.T, status int, response string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.key = r.Header.Get("X-Goog-Api-Key")
		captured.query = r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured.body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return server, captured
}

const translated = `{"data":{"translations":[{"translatedText":"Please fix the failing test","detectedSourceLanguage":"de"}]}}`

func TestTranslateReturnsTheEnglishTextForTheDraft(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK, translated)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	english, err := client.Translate(context.Background(), "Bitte behebe den fehlschlagenden Test")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if english != "Please fix the failing test" {
		t.Errorf("Translate returned %q, want the translated text", english)
	}
	if captured.body["q"] != "Bitte behebe den fehlschlagenden Test" {
		t.Errorf("request sent %v, want the draft", captured.body["q"])
	}
}

// The key belongs in a header. In the query string it would be written down by
// every proxy and server log on the way.
func TestTheKeyTravelsInAHeaderRatherThanTheQueryString(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK, translated)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	if _, err := client.Translate(context.Background(), "Bitte behebe den Test"); err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if captured.key != "key-123" {
		t.Errorf("request carried the key as %q, want it in X-Goog-Api-Key", captured.key)
	}
	if strings.Contains(captured.query, "key-123") {
		t.Errorf("the key is in the query string: %q", captured.query)
	}
}

// Google's default is HTML, which comes back with escaped entities.
func TestADraftIsSentAndReturnedAsPlainText(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK,
		`{"data":{"translations":[{"translatedText":"Don't fix what isn't broken"}]}}`)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	english, err := client.Translate(context.Background(), "Repariere nichts, was nicht kaputt ist")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if captured.body["format"] != "text" {
		t.Errorf("request asked for format %v, want text", captured.body["format"])
	}
	if english != "Don't fix what isn't broken" {
		t.Errorf("Translate returned %q, want the apostrophes as characters", english)
	}
}

// An answer that arrives HTML-escaped anyway is unescaped rather than passed on.
func TestEscapedEntitiesInTheAnswerAreTurnedBackIntoCharacters(t *testing.T) {
	t.Parallel()
	server, _ := serverReturning(t, http.StatusOK,
		`{"data":{"translations":[{"translatedText":"Don&#39;t use &lt;br&gt; &amp; friends"}]}}`)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	english, err := client.Translate(context.Background(), "Nutze keine Entities")
	if err != nil {
		t.Fatalf("Translate returned unexpected error: %v", err)
	}
	if english != "Don't use <br> & friends" {
		t.Errorf("Translate returned %q, want the entities decoded", english)
	}
}

// The language is configured once, in whatever spelling the other service wanted.
func TestTheTargetLanguageIsTranslatedIntoWhatGoogleUnderstands(t *testing.T) {
	t.Parallel()
	for _, wanted := range []struct{ configured, sent string }{
		{"EN-US", "en"},
		{"EN-GB", "en-GB"},
		{"en", "en"},
		{"PT-BR", "pt-BR"},
		{"ZH", "zh"},
		{"", "en"},
	} {
		server, captured := serverReturning(t, http.StatusOK, translated)
		client := google.New("key-123",
			google.WithEndpoint(server.URL), google.WithTargetLanguage(wanted.configured))

		if _, err := client.Translate(context.Background(), "Bitte behebe den Test"); err != nil {
			t.Fatalf("Translate returned unexpected error: %v", err)
		}
		if captured.body["target"] != wanted.sent {
			t.Errorf("%q was sent as %v, want %q", wanted.configured, captured.body["target"], wanted.sent)
		}
	}
}

func TestAServiceErrorIsReportedWithWhatItSaid(t *testing.T) {
	t.Parallel()
	server, _ := serverReturning(t, http.StatusForbidden,
		`{"error":{"code":403,"message":"API key not valid","status":"PERMISSION_DENIED"}}`)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	_, err := client.Translate(context.Background(), "Bitte behebe den Test")
	if err == nil {
		t.Fatal("Translate returned no error for a refused request")
	}
	if !strings.Contains(err.Error(), "API key not valid") {
		t.Errorf("Translate returned %v, want what the service said", err)
	}
}

func TestAnEmptyAnswerIsAnErrorRatherThanAnEmptyPrompt(t *testing.T) {
	t.Parallel()
	server, _ := serverReturning(t, http.StatusOK, `{"data":{"translations":[]}}`)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	if _, err := client.Translate(context.Background(), "Bitte behebe den Test"); err == nil {
		t.Error("Translate returned no error for an answer with no translation in it")
	}
}

func TestADraftTooLargeForTheServiceIsRefusedBeforeItIsSent(t *testing.T) {
	t.Parallel()
	server, captured := serverReturning(t, http.StatusOK, translated)
	client := google.New("key-123", google.WithEndpoint(server.URL))

	_, err := client.Translate(context.Background(), strings.Repeat("z", 40_000))
	if err == nil {
		t.Fatal("Translate returned no error for a draft past the service limit")
	}
	if captured.body != nil {
		t.Error("the oversized draft was sent anyway")
	}
}
