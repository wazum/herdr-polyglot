package translation

import "strings"

// piece is one sentence plus the whitespace that followed it, so a draft can be
// put back together exactly as it was written.
type piece struct {
	text      string
	separator string
}

func splitSentences(draft string) []piece {
	var pieces []piece
	runes := []rune(draft)

	start := 0
	for index := 0; index < len(runes); index++ {
		// A line break always ends a piece and belongs to the separator; a
		// terminator ends one only when the sentence really stops there.
		end := index
		switch {
		case runes[index] == '\n':
		case endsSentence(runes, index):
			end = index + 1
		default:
			continue
		}

		separatorEnd := end
		for separatorEnd < len(runes) && isSeparator(runes[separatorEnd]) {
			separatorEnd++
		}

		pieces = append(pieces, piece{
			text:      string(runes[start:end]),
			separator: string(runes[end:separatorEnd]),
		})
		start, index = separatorEnd, separatorEnd-1
	}

	if start < len(runes) {
		pieces = append(pieces, piece{text: string(runes[start:])})
	}
	return pieces
}

func endsSentence(runes []rune, index int) bool {
	switch runes[index] {
	case '.', '!', '?', '…':
		// A terminator only ends a sentence when what follows is not more of it,
		// which keeps "z.B." and "1.2" in one piece.
		return index+1 >= len(runes) || isSeparator(runes[index+1])
	default:
		return false
	}
}

func isSeparator(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// surroundings is everything except the piece being translated: free context
// that lets a service translate one sentence as part of the whole draft.
func surroundings(pieces []piece, index int) string {
	var rest strings.Builder
	for other, piece := range pieces {
		if other == index {
			continue
		}
		rest.WriteString(piece.text)
		rest.WriteString(piece.separator)
	}
	return strings.TrimSpace(rest.String())
}

func join(pieces []piece, translated []string) string {
	var whole strings.Builder
	for index, piece := range pieces {
		whole.WriteString(translated[index])
		whole.WriteString(piece.separator)
	}
	return whole.String()
}
