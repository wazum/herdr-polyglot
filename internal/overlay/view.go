package overlay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/translation"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

func (m Model) View() string {
	draft := m.styles.draftBox.Width(m.width).Render(m.draft.View())
	line := lipgloss.Width(draft)

	parts := []string{m.header(line), draft}
	if m.showsEnglish() {
		parts = append(parts, m.englishPane())
	}
	parts = append(parts, m.footer(line))

	// Herdr already draws a frame around the popup; a second one inside it only
	// takes room from the draft.
	return strings.Join(parts, "\n")
}

func (m Model) englishPane() string {
	body := m.styles.placeholder.Render("…")
	switch {
	case m.previewError != nil:
		body = m.styles.danger.Render("✗ " + m.previewError.Error())
	case m.preview != "":
		body = m.styles.text.Render(m.preview)
		if !m.previewIsCurrent() {
			body = m.styles.placeholder.Render(m.preview)
		}
	}
	return m.styles.englishBox.Width(m.width).Render(body)
}

// Herdr writes the plugin's name on the popup frame, so the heading is only the
// badge saying what will happen. It starts one column in, under that name, and
// is cut rather than allowed to wrap onto the draft.
func (m Model) header(line int) string {
	return lipgloss.NewStyle().MaxWidth(line - 1).Render(" " + m.badge())
}

// The badge is joined from rendered pieces: the pulse carries its own colour,
// which a single Render around everything would cut short.
func (m Model) badge() string {
	pieces := []string{m.styles.badge.Render(m.options.Service + " → " + m.options.Language)}

	switch {
	case m.options.Live && m.options.Pulse:
		pieces = append(pieces,
			m.styles.badge.Render("·"), m.pulseGlyph(), m.styles.badge.Render("live"))
	case m.options.Live:
		pieces = append(pieces, m.styles.badge.Render("· live"))
	default:
		// Saying so is worth a word: ctrl+l is what turns it on.
		pieces = append(pieces, m.styles.badge.Render("· live off"))
	}

	pieces = append(pieces, m.styles.badge.Render("· "+m.whatHappens()))
	if m.resumed {
		pieces = append(pieces, m.styles.badge.Render("· resumed draft"))
	}
	if m.spentKnown {
		// Services bill translation by the character, so that is what the number
		// counts, and it is the month's allowance it counts against.
		pieces = append(pieces,
			m.styles.badge.Render("· "+spentAsWords(m.spent)+" chars this month"))
	}
	return strings.Join(pieces, m.styles.badge.Render(" "))
}

// The heading has room to say what will happen in words; the footer, which has
// to hold every key, names the same thing in one.
func (m Model) whatHappens() string {
	if m.delivery == promptflow.Typing {
		return "fills the input"
	}
	return "sends to agent"
}

func (m Model) destination() string {
	if m.delivery == promptflow.Typing {
		return "fill"
	}
	return "send"
}

func (m Model) otherDestination() string {
	if m.delivery == promptflow.Typing {
		return "send"
	}
	return "fill"
}

// The mode sits at the end of the line, next to the key that changes it.
func (m Model) footer(line int) string {
	mode := ""
	if m.draft.Modal() {
		mode = m.styles.mode.Render(m.draft.Mode().String())
	}

	// One column in on both sides, like the heading.
	inner := line - 1

	switch {
	case m.failure != nil:
		return spread(" "+m.styles.danger.Render("✗ "+m.failure.Error()),
			m.styles.hint.Render("ctrl+c close"), inner)
	case m.draftIsTooLong():
		return spread(
			" "+m.styles.danger.Render(fmt.Sprintf("⚠ %d characters", len([]rune(m.draft.Value()))))+
				m.styles.hint.Render(" — this box is for prompts you write, not files you paste"),
			m.styles.hint.Render("ctrl+u discard"), inner)
	case m.stage == confirming:
		return spread(
			" "+m.styles.key.Render("ctrl+d")+m.styles.hint.Render(" send this")+
				m.styles.hint.Render(" · ")+m.styles.key.Render("esc")+
				m.styles.hint.Render(" keep writing"),
			m.styles.badge.Render("read it first"), inner)
	case m.stage == translating:
		return spread(" "+m.spinner.View()+m.styles.accent.Render(" translating …"), mode, inner)
	default:
		return spread(" "+m.keyHints(), mode, inner)
	}
}

// Every key fits on the line as long as each is named in one word, so there is
// nothing to go looking for. The vim bindings are the exception, and they are in
// the readme rather than the footer.
func (m Model) keyHints() string {
	shown := [][2]string{
		{"ctrl+d", m.destination()},
		{"ctrl+r", "→ " + m.otherDestination()},
		{"ctrl+l", "live"},
	}

	// An arrow reads as "takes you to", which a bare mode name does not.
	switch {
	case m.draft.Modal() && m.draft.Mode() == vimarea.Normal:
		shown = append(shown, [2]string{"i", "→ insert"}, [2]string{"q", "close"})
	case m.draft.Modal():
		shown = append(shown, [2]string{"ctrl+u", "clear"}, [2]string{"esc", "→ normal"})
	default:
		shown = append(shown, [2]string{"ctrl+u", "clear"}, [2]string{"esc", "close"})
	}

	hints := make([]string, 0, len(shown))
	for _, hint := range shown {
		hints = append(hints, m.styles.key.Render(hint[0])+m.styles.hint.Render(" "+hint[1]))
	}
	return strings.Join(hints, m.styles.hint.Render(" · "))
}

// spread pins left to the start of the line and right to its end.
func spread(left, right string, line int) string {
	if right == "" {
		return left
	}

	gap := line - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

// Short enough for a header: 12.3k/1M.
func spentAsWords(spent translation.Usage) string {
	return compactCount(spent.Used) + "/" + compactCount(spent.Limit)
}

func compactCount(count int64) string {
	switch {
	case count >= 1_000_000:
		return trimZero(float64(count)/1_000_000) + "M"
	case count >= 1_000:
		return trimZero(float64(count)/1_000) + "k"
	default:
		return strconv.FormatInt(count, 10)
	}
}

func trimZero(value float64) string {
	rendered := strconv.FormatFloat(value, 'f', 1, 64)
	return strings.TrimSuffix(rendered, ".0")
}
