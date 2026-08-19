package main

import (
	"fmt"

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
// default service; no key and no choice means none at all, and the popup is then
// the draft box it always was.
func chooseService(settings *config.Settings) service {
	registry := services()

	name := settings.Provider
	if name == "" {
		name = translation.OffProvider{}.Name()
		if settings.Options.APIKey != "" {
			name = registry.Default()
		}
	}

	translator, err := registry.Translator(name, settings.Options)
	switch {
	case err == nil:
		return service{
			name:       name,
			translator: translator,
			translates: name != translation.OffProvider{}.Name(),
		}
	case settings.ConfigFile != "":
		err = fmt.Errorf("%s: %w; configure it in %s", name, err, settings.ConfigFile)
	default:
		err = fmt.Errorf("%s: %w", name, err)
	}

	return service{
		name:       name,
		translator: translation.Broken{Why: err},
		translates: true,
		trouble:    err,
	}
}
