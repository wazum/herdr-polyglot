package deepl

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

type Provider struct{}

func (Provider) Name() string { return "deepl" }

func (Provider) New(options translation.Options) (translation.Translator, error) {
	if options.APIKey == "" {
		return nil, errors.New("no API key")
	}
	if err := requireSecureEndpoint(options.Endpoint); err != nil {
		return nil, err
	}

	var clientOptions []Option
	if options.TargetLanguage != "" {
		clientOptions = append(clientOptions, WithTargetLanguage(options.TargetLanguage))
	}
	if options.Endpoint != "" {
		clientOptions = append(clientOptions, WithEndpoint(options.Endpoint))
	}
	return New(options.APIKey, clientOptions...), nil
}

// requireSecureEndpoint keeps the API key off the wire in clear. A loopback
// address stays allowed so a test server needs no certificate.
func requireSecureEndpoint(endpoint string) error {
	if endpoint == "" {
		return nil
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint %q is not a URL: %w", endpoint, err)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopback(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("endpoint %q must use https, or the API key travels in clear", endpoint)
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}
