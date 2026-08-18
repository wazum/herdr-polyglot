package translation

import (
	"context"
	"errors"
	"strings"
	"sync"
)

const (
	// contextSentences is how much of what came before a sentence is sent along
	// and remembered with it. One sentence is enough to settle most ambiguity,
	// and keeping the window small means editing early in a draft does not
	// invalidate everything after it.
	contextSentences = 1
	// maxRemembered bounds the store: a long session would otherwise keep every
	// sentence it ever translated.
	maxRemembered = 256
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

	mu    sync.Mutex
	known map[string]string
	// remembered is insertion order, so the oldest entry goes first.
	remembered []string
	inflight   map[string]*call
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
	for index, piece := range pieces {
		if strings.TrimSpace(piece.text) == "" {
			translated[index] = piece.text
			continue
		}

		result, err := s.translatePiece(ctx, piece, contextBefore(pieces, index), index == len(pieces)-1)
		if err != nil {
			return "", err
		}
		translated[index] = result
	}

	return join(pieces, translated), nil
}

func (s *segmented) translatePiece(ctx context.Context, piece piece, preceding string, isTail bool) (string, error) {
	return s.translateOnce(ctx, request{
		key:       preceding + "\x00" + piece.text,
		text:      piece.text,
		preceding: preceding,
		asTail:    isTail && !piece.finished,
	})
}

type request struct {
	key       string
	text      string
	preceding string
	asTail    bool
}

// translateOnce keeps overlapping previews from paying twice for one sentence:
// the second caller waits for the first instead of sending its own request. It
// waits on its own terms, though — a caller that gave up must not take this one
// down with it, which would fail a send that merely shared a sentence with an
// abandoned preview.
func (s *segmented) translateOnce(ctx context.Context, wanted request) (string, error) {
	for {
		s.mu.Lock()
		if cached, found := s.cached(wanted.key); found {
			s.mu.Unlock()
			return cached, nil
		}
		running, found := s.inflight[wanted.key]
		if !found {
			running = &call{done: make(chan struct{})}
			s.inflight[wanted.key] = running
			s.mu.Unlock()
			return s.send(ctx, wanted, running)
		}
		s.mu.Unlock()

		select {
		case <-running.done:
			switch {
			case running.err == nil:
				return running.text, nil
			case gaveUp(running.err):
				// Whoever asked first walked away; ask again for ourselves.
				continue
			default:
				return "", running.err
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func (s *segmented) send(ctx context.Context, wanted request, running *call) (string, error) {
	running.text, running.err = s.callTranslator(ctx, wanted.text, wanted.preceding)

	// Remember and step aside under one lock, so nobody starts the same request
	// in the gap between the two.
	s.mu.Lock()
	if running.err == nil {
		s.keep(wanted.key, running.text, wanted.asTail)
	}
	delete(s.inflight, wanted.key)
	s.mu.Unlock()

	close(running.done)
	return running.text, running.err
}

func gaveUp(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (s *segmented) callTranslator(ctx context.Context, text, preceding string) (string, error) {
	if contextual, ok := s.translator.(ContextualTranslator); ok {
		return contextual.TranslateWithContext(ctx, text, preceding)
	}
	return s.translator.Translate(ctx, text)
}

// cached expects the lock to be held.
func (s *segmented) cached(key string) (string, bool) {
	if cached, found := s.known[key]; found {
		return cached, true
	}
	if key == s.tailKey {
		return s.tailText, true
	}
	return "", false
}

// keep expects the lock to be held.
func (s *segmented) keep(key, translated string, asTail bool) {
	if asTail {
		s.tailKey, s.tailText = key, translated
		return
	}

	if _, known := s.known[key]; !known {
		s.remembered = append(s.remembered, key)
		if len(s.remembered) > maxRemembered {
			delete(s.known, s.remembered[0])
			s.remembered = s.remembered[1:]
		}
	}
	s.known[key] = translated
}

// contextBefore is the handful of sentences in front of one, which is what the
// service is told about its meaning.
func contextBefore(pieces []piece, index int) string {
	from := max(index-contextSentences, 0)

	var before strings.Builder
	for _, piece := range pieces[from:index] {
		before.WriteString(piece.text)
		before.WriteString(piece.separator)
	}
	return strings.TrimSpace(before.String())
}
