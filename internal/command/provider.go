package command

import (
	"errors"

	"github.com/wazum/herdr-polyglot/internal/translation"
)

type Provider struct{}

func (Provider) Name() string { return "cmd" }

func (Provider) New(options translation.Options) (translation.Translator, error) {
	if options.Command == "" {
		return nil, errors.New("no command to run: set HERDR_POLYGLOT_COMMAND")
	}
	return New(options.Command, WithTargetLanguage(options.TargetLanguage)), nil
}
