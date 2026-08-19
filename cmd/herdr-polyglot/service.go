package main

import (
	"errors"
	"fmt"

	"github.com/wazum/herdr-polyglot/internal/command"
	"github.com/wazum/herdr-polyglot/internal/config"
	"github.com/wazum/herdr-polyglot/internal/translation"
)

type service struct {
	name       string
	translator translation.Translator
	// translates says the popup has something to translate with, so it offers the
	// keys and the second panel that go with it.
	translates bool
	// trouble is a service that was asked for and could not be built, which the
	// popup says out loud.
	trouble error
}

// chooseService decides what the draft goes through. A key on its own means the
// default service, a command on its own the program it names; no key, no command
// and no choice means none at all, and the popup is then the draft box it always
// was.
func chooseService(settings *config.Settings) service {
	registry := services()

	name, err := chosenName(settings, registry)
	if err == nil {
		var translator translation.Translator
		translator, err = registry.Translator(name, settings.Options)
		if err == nil {
			return service{
				name:       name,
				translator: translator,
				translates: name != translation.OffProvider{}.Name(),
			}
		}
	}

	if settings.ConfigFile != "" {
		err = fmt.Errorf("%w; configure it in %s", err, settings.ConfigFile)
	}
	return service{
		name:       name,
		translator: translation.Broken{Why: err},
		translates: true,
		trouble:    err,
	}
}

// A key and a command are two answers to the same question, and picking one of
// them would send the draft somewhere it was not meant to go.
func chosenName(settings *config.Settings, registry *translation.Registry) (string, error) {
	if settings.Provider != "" {
		return settings.Provider, nil
	}

	hasKey, hasCommand := settings.Options.APIKey != "", settings.Options.Command != ""
	switch {
	case hasKey && hasCommand:
		return "", errors.New(
			"there is both an API key and a command, so it is unclear which translates: " +
				"choose one with HERDR_POLYGLOT_PROVIDER")
	case hasCommand:
		return command.Provider{}.Name(), nil
	case hasKey:
		return registry.Default(), nil
	default:
		return translation.OffProvider{}.Name(), nil
	}
}
