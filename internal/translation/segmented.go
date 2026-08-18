package translation

import (
	"context"
	"strings"
	"sync"
)

// ContextualTranslator can be told what surrounds a piece of text. DeepL uses
// it to translate the piece in context without billing for the surroundings.
type ContextualTranslator interface {
	TranslateWithContext(ctx context.Context, text, surrounding string) (string, error)
}

// segmented translates a draft sentence by sentence and remembers what each
// sentence translated to. While a draft is written, only the sentence being
// typed is sent; everything before it is already known.
type segmented struct {
	translator Translator

	mu    sync.Mutex
	known map[string]string
}

func Segmented(translator Translator) Translator {
	return &segmented{translator: translator, known: map[string]string{}}
}

func (s *segmented) Translate(ctx context.Context, draft string) (string, error) {
	pieces := splitSentences(draft)

	translated := make([]string, len(pieces))
	for index, piece := range pieces {
		if strings.TrimSpace(piece.text) == "" {
			translated[index] = piece.text
			continue
		}

		result, err := s.translatePiece(ctx, piece.text, surroundings(pieces, index))
		if err != nil {
			return "", err
		}
		translated[index] = result
	}

	return join(pieces, translated), nil
}

func (s *segmented) translatePiece(ctx context.Context, text, surrounding string) (string, error) {
	if cached, found := s.lookup(text); found {
		return cached, nil
	}

	result, err := s.callTranslator(ctx, text, surrounding)
	if err != nil {
		return "", err
	}

	s.remember(text, result)
	return result, nil
}

func (s *segmented) callTranslator(ctx context.Context, text, surrounding string) (string, error) {
	if contextual, ok := s.translator.(ContextualTranslator); ok {
		return contextual.TranslateWithContext(ctx, text, surrounding)
	}
	return s.translator.Translate(ctx, text)
}

func (s *segmented) lookup(text string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cached, found := s.known[text]
	return cached, found
}

func (s *segmented) remember(text, translated string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.known[text] = translated
}
