package deepl

import (
	"errors"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

type Provider struct{}

func (Provider) Name() string { return "deepl" }

func (Provider) New(options translation.Options) (translation.Translator, error) {
	if options.APIKey == "" {
		return nil, errors.New("no API key")
	}

	clientOptions := []Option{}
	if options.TargetLanguage != "" {
		clientOptions = append(clientOptions, WithTargetLanguage(options.TargetLanguage))
	}
	if options.Endpoint != "" {
		clientOptions = append(clientOptions, WithEndpoint(options.Endpoint))
	}
	return New(options.APIKey, clientOptions...), nil
}
