package translation

import (
	"context"
	"strings"
	"sync"
)

// ContextualTranslator can be told what comes before a piece of text. DeepL uses
// it to translate the piece in context without billing for the surroundings.
type ContextualTranslator interface {
	TranslateWithContext(ctx context.Context, text, preceding string) (string, error)
}

// segmented translates a draft sentence by sentence and remembers what each
// sentence translated to. While a draft is written, only the sentence being
// typed is sent; everything before it is already known.
//
// A sentence is remembered together with what preceded it, because that is what
// the service was told about its meaning. Typing at the end therefore keeps the
// earlier sentences, while editing an earlier one retranslates what follows.
type segmented struct {
	translator Translator

	mu       sync.Mutex
	known    map[string]string
	inflight map[string]*call
	// tail is the sentence still being written. It is kept in one slot instead
	// of the map, so a long session does not collect an entry per keystroke.
	tailKey, tailText string
}

type call struct {
	done chan struct{}
	text string
	err  error
}

func Segmented(translator Translator) Translator {
	return &segmented{
		translator: translator,
		known:      map[string]string{},
		inflight:   map[string]*call{},
	}
}

func (s *segmented) Translate(ctx context.Context, draft string) (string, error) {
	pieces := splitSentences(draft)

	translated := make([]string, len(pieces))
	preceding := ""
	for index, piece := range pieces {
		if strings.TrimSpace(piece.text) == "" {
			translated[index] = piece.text
			preceding = join(pieces[:index+1], textsOf(pieces[:index+1]))
			continue
		}

		result, err := s.translatePiece(ctx, piece, preceding, index == len(pieces)-1)
		if err != nil {
			return "", err
		}
		translated[index] = result
		preceding = strings.TrimSpace(preceding + " " + piece.text)
	}

	return join(pieces, translated), nil
}

func (s *segmented) translatePiece(ctx context.Context, piece piece, preceding string, isTail bool) (string, error) {
	key := preceding + "\x00" + piece.text

	if cached, found := s.lookup(key); found {
		return cached, nil
	}

	result, err := s.translateOnce(ctx, key, piece.text, preceding)
	if err != nil {
		return "", err
	}

	s.remember(key, result, isTail && !piece.finished)
	return result, nil
}

// translateOnce keeps overlapping previews from paying twice for one sentence:
// the second caller waits for the first instead of sending its own request.
func (s *segmented) translateOnce(ctx context.Context, key, text, preceding string) (string, error) {
	s.mu.Lock()
	if running, found := s.inflight[key]; found {
		s.mu.Unlock()
		<-running.done
		return running.text, running.err
	}
	running := &call{done: make(chan struct{})}
	s.inflight[key] = running
	s.mu.Unlock()

	running.text, running.err = s.callTranslator(ctx, text, preceding)
	close(running.done)

	s.mu.Lock()
	delete(s.inflight, key)
	s.mu.Unlock()

	return running.text, running.err
}

func (s *segmented) callTranslator(ctx context.Context, text, preceding string) (string, error) {
	if contextual, ok := s.translator.(ContextualTranslator); ok {
		return contextual.TranslateWithContext(ctx, text, preceding)
	}
	return s.translator.Translate(ctx, text)
}

func (s *segmented) lookup(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cached, found := s.known[key]; found {
		return cached, true
	}
	if key == s.tailKey {
		return s.tailText, true
	}
	return "", false
}

func (s *segmented) remember(key, translated string, asTail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if asTail {
		s.tailKey, s.tailText = key, translated
		return
	}
	s.known[key] = translated
}

func textsOf(pieces []piece) []string {
	texts := make([]string, len(pieces))
	for index, piece := range pieces {
		texts[index] = piece.text
	}
	return texts
}
