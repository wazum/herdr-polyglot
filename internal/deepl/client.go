// Package deepl translates drafts through the DeepL API.
package deepl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	proEndpoint  = "https://api.deepl.com/v2/translate"
	freeEndpoint = "https://api-free.deepl.com/v2/translate"
	// DeepL marks free-tier keys with this suffix, and they only work on the free host.
	freeKeySuffix = ":fx"

	defaultTargetLanguage = "EN-US"
	defaultTimeout        = 15 * time.Second

	maxErrorBodyBytes = 2 << 10
	// A translated prompt is small; anything larger is a broken or hostile
	// endpoint rather than a translation.
	maxResponseBytes = 1 << 20
	// DeepL refuses a request larger than 128 KiB, so say so here rather than
	// send a draft across the network to be turned away.
	maxRequestBytes = 128 << 10
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
	return func(c *Client) { c.targetLanguage = language }
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithSecureOnly refuses a redirect that would carry the key off https. Go keeps
// the Authorization header when a host redirects to itself, downgrade included.
func WithSecureOnly() Option {
	return func(c *Client) {
		c.httpClient.CheckRedirect = func(request *http.Request, _ []*http.Request) error {
			if request.URL.Scheme != "https" {
				return fmt.Errorf("refusing a redirect to %s: the API key travels with it",
					request.URL.Scheme)
			}
			return nil
		}
	}
}

func New(apiKey string, options ...Option) *Client {
	client := &Client{
		httpClient:     &http.Client{Timeout: defaultTimeout},
		endpoint:       endpointFor(apiKey),
		apiKey:         apiKey,
		targetLanguage: defaultTargetLanguage,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func endpointFor(apiKey string) string {
	if strings.HasSuffix(apiKey, freeKeySuffix) {
		return freeEndpoint
	}
	return proEndpoint
}

func (c *Client) Endpoint() string {
	return c.endpoint
}

type translateRequest struct {
	Text               []string `json:"text"`
	TargetLang         string   `json:"target_lang"`
	PreserveFormatting bool     `json:"preserve_formatting"`
	// Context informs the translation without being translated or billed.
	Context string `json:"context,omitempty"`
}

type translateResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
}

func (c *Client) Translate(ctx context.Context, draft string) (string, error) {
	return c.TranslateWithContext(ctx, draft, "")
}

func (c *Client) TranslateWithContext(ctx context.Context, draft, surrounding string) (string, error) {
	body, err := json.Marshal(translateRequest{
		Text:               []string{draft},
		TargetLang:         c.targetLanguage,
		PreserveFormatting: true,
		Context:            surrounding,
	})
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}
	if len(body) > maxRequestBytes {
		return "", fmt.Errorf("draft is too long to translate: %d of %d bytes",
			len(body), maxRequestBytes)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	request.Header.Set("Authorization", "DeepL-Auth-Key "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("calling DeepL: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepL rejected the request: %s: %s",
			response.Status, explanation(response.Body))
	}

	var decoded translateResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if len(decoded.Translations) == 0 {
		return "", errors.New("DeepL returned no translation")
	}
	return decoded.Translations[0].Text, nil
}

func explanation(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
		return "no details"
	}

	var problem struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &problem); err == nil && problem.Message != "" {
		return problem.Message
	}
	if trimmed := strings.TrimSpace(string(raw)); trimmed != "" {
		return trimmed
	}
	return "no details"
}
