// Package google translates through Google Cloud Translation. It is one service
// behind the translation ports, and knows nothing about the overlay.
package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// v2 takes a plain API key. v3 would need a service account and a project id,
	// which is a different kind of credential than a plugin config file holds.
	defaultEndpoint = "https://translation.googleapis.com/language/translate/v2"

	defaultTargetLanguage = "en"
	defaultTimeout        = 15 * time.Second

	maxErrorBodyBytes = 2 << 10
	maxResponseBytes  = 1 << 20
	// The service refuses a larger request, so say so here rather than send a
	// draft across the network to be turned away.
	maxRequestBytes = 30 << 10
)

type Client struct {
	httpClient     *http.Client
	endpoint       string
	apiKey         string
	targetLanguage string
}

type Option func(*Client)

func WithEndpoint(endpoint string) Option {
	return func(c *Client) { c.endpoint = endpoint }
}

func WithTargetLanguage(language string) Option {
	return func(c *Client) { c.targetLanguage = languageFor(language) }
}

// refuseInsecureRedirect keeps the key from following a redirect off https. Go
// carries the credential header when a host redirects to itself, downgrade
// included, so this is installed for every client rather than asked for.
func refuseInsecureRedirect(request *http.Request, _ []*http.Request) error {
	if request.URL.Scheme != "https" {
		return fmt.Errorf("refusing a redirect to %s: the API key travels with it",
			request.URL.Scheme)
	}
	return nil
}

func New(apiKey string, options ...Option) *Client {
	client := &Client{
		httpClient:     &http.Client{Timeout: defaultTimeout},
		endpoint:       defaultEndpoint,
		apiKey:         apiKey,
		targetLanguage: defaultTargetLanguage,
	}
	for _, option := range options {
		option(client)
	}
	// After the options, so no way of building a client can leave the key free to
	// follow a redirect into the clear.
	client.httpClient.CheckRedirect = refuseInsecureRedirect
	return client
}

func (c *Client) Endpoint() string {
	return c.endpoint
}

type translateRequest struct {
	Text string `json:"q"`
	// Target is the language wanted; the source is left to the service to detect.
	Target string `json:"target"`
	// Format keeps the draft plain: the service otherwise treats it as HTML and
	// answers with escaped entities.
	Format string `json:"format"`
}

type translateResponse struct {
	Data struct {
		Translations []struct {
			Text string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
}

func (c *Client) Translate(ctx context.Context, draft string) (string, error) {
	body, err := json.Marshal(translateRequest{
		Text:   draft,
		Target: c.targetLanguage,
		Format: "text",
	})
	if err != nil {
		return "", fmt.Errorf("preparing the translation request: %w", err)
	}
	if len(body) > maxRequestBytes {
		return "", fmt.Errorf("the draft is %d bytes, more than the %d the service takes at once",
			len(body), maxRequestBytes)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("preparing the translation request: %w", err)
	}
	// In the query string the key would be written down by every proxy and log on
	// the way; a header is not.
	request.Header.Set("X-Goog-Api-Key", c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("asking the service to translate: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the service refused the draft: %s%s",
			response.Status, explanation(response.Body))
	}

	var decoded translateResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&decoded); err != nil {
		return "", fmt.Errorf("reading the translation: %w", err)
	}
	if len(decoded.Data.Translations) == 0 {
		return "", errors.New("the service answered without a translation")
	}
	// Entities come back even from a plain-text request when the draft held one.
	return html.UnescapeString(decoded.Data.Translations[0].Text), nil
}

// languageFor takes the language in whatever spelling another service wanted and
// says it the way this one does: a plain code, or a code with the region kept
// where the region is the point.
func languageFor(language string) string {
	if language == "" {
		return defaultTargetLanguage
	}

	code, region, hasRegion := strings.Cut(strings.TrimSpace(language), "-")
	code = strings.ToLower(code)
	if !hasRegion {
		return code
	}

	region = strings.ToUpper(region)
	if regionMatters[code+"-"+region] {
		return code + "-" + region
	}
	return code
}

// The service takes a plain code for most languages and a regional one only for
// these, where the two are far enough apart to be worth asking for.
var regionMatters = map[string]bool{
	"en-GB": true,
	"pt-BR": true,
	"pt-PT": true,
	"zh-CN": true,
	"zh-TW": true,
	"fr-CA": true,
	"es-MX": true,
}

func explanation(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil || len(raw) == 0 {
		return ""
	}

	var refusal struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &refusal); err == nil && refusal.Error.Message != "" {
		return " — " + refusal.Error.Message
	}
	return " — " + strings.TrimSpace(string(raw))
}
