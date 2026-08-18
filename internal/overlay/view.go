package overlay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/wazum/herdr-polyglot/internal/promptflow"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

func (m Model) View() string {
	draft := m.styles.draftBox.Width(m.width).Render(m.draftBody())
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

// The mark is only ever drawn over space nobody is using, so it goes as soon as
// there is a draft to read.
func (m Model) draftBody() string {
	body := m.draft.View()
	if m.options.Logo && m.draft.Value() == "" {
		body = m.sign(body)
	}
	first, visible, total := m.draftScroll()
	return m.scrolled(body, first, visible, total)
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
	// The box holds the rows it holds: a longer translation scrolls to its end
	// rather than growing the box and pushing the footer off the pane.
	rows := englishRows - 2
	total := rowsOf(body, m.contentWidth())
	shown := lastRows(body, m.contentWidth(), rows)
	return m.styles.englishBox.Width(m.width).
		Render(m.scrolled(shown, max(total-rows, 0), rows, total))
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

	// A glyph and the word, always the same width, so switching live translation
	// on or off does not shift everything after it along the line.
	pieces = append(pieces,
		m.styles.badge.Render("·"),
		m.liveState(),
		m.styles.badge.Render("· "+m.whatHappens()),
	)
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

// A circle that breathes while translating, or a struck-through word when live
// translation is off. Both are the same width, so nothing after them moves.
func (m Model) liveState() string {
	if m.options.Live {
		glyph := m.styles.bright.Render("●")
		if m.options.Pulse {
			glyph = m.pulseGlyph()
		}
		return glyph + m.styles.badge.Render(" live")
	}
	return m.styles.badge.Render("✘") + m.styles.off.Render(" live")
}

// The heading has room to say what will happen in words; the footer, which has
// to hold every key, names the same thing in one. Both readings are padded to
// one width so switching between them moves nothing.
func (m Model) whatHappens() string {
	const widest = len("fills the input")
	if m.delivery == promptflow.Typing {
		return "fills the input"
	}
	return fmt.Sprintf("%-*s", widest, "sends to agent")
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
	case m.notice != nil:
		return spread(" "+m.styles.danger.Render("✗ "+m.notice.Error()),
			m.styles.key.Render("esc")+m.styles.hint.Render(" dismiss"), inner)
	case m.hint != "":
		return spread(" "+m.styles.badge.Render(m.hint),
			m.styles.key.Render("esc")+m.styles.hint.Render(" dismiss"), inner)
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
		return spread(" "+m.keyHints(roomBeside(mode, inner)), mode, inner)
	}
}

// roomBeside is what a line has left for the hints once the mode has its place.
func roomBeside(mode string, inner int) int {
	room := inner - 1
	if mode != "" {
		room -= lipgloss.Width(mode) + 1
	}
	return room
}

// Every key fits on the line as long as each is named in one word, so there is
// nothing to go looking for. The vim bindings are the exception, and they are in
// the readme rather than the footer.
func (m Model) keyHints(room int) string {
	shown := [][2]string{
		{"alt+enter", m.destination()},
		{"ctrl+r", "→ " + m.otherDestination()},
		{"ctrl+l", "live"},
	}

	// An arrow reads as "takes you to", which a bare mode name does not.
	switch {
	case m.draft.Modal() && m.draft.Mode() == vimarea.Normal:
		shown = append(shown, [2]string{"i", "→ insert"}, [2]string{"esc", "close"})
	case m.draft.Modal():
		shown = append(shown, [2]string{"ctrl+u", "clear"}, [2]string{"esc", "→ normal"})
	default:
		shown = append(shown, [2]string{"ctrl+u", "clear"}, [2]string{"esc", "close"})
	}

	// A pane too narrow for every key shows the ones it has room for. Half a key
	// name is worth less than none.
	separator := m.styles.hint.Render(" · ")
	line, width := "", 0
	for _, hint := range shown {
		drawn := m.styles.key.Render(hint[0]) + m.styles.hint.Render(" "+hint[1])
		needed := lipgloss.Width(drawn)
		if line != "" {
			needed += lipgloss.Width(separator)
		}
		if width+needed > room {
			break
		}
		if line != "" {
			line += separator
		}
		line += drawn
		width += needed
	}
	return line
}

// lastRows keeps the end of the text, where the newest writing is.
func lastRows(text string, width, rows int) string {
	if width < 1 || rows < 1 {
		return text
	}

	lines := strings.Split(wrapped(text, width), "\n")
	if len(lines) <= rows {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-rows:], "\n")
}

// spread pins left to the start of the line and right to its end.
func spread(left, right string, line int) string {
	// Nothing here may outgrow the line: a wrapped footer would push the draft up
	// and the popup out of the pane.
	if right == "" {
		return cutTo(left, line)
	}

	gap := line - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		// Too narrow for both, and the keys are worth more than the mode.
		return cutTo(left, line)
	}
	return left + strings.Repeat(" ", gap) + right
}

func cutTo(text string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(text)
}

// Short enough for a header: 12.3k/1M.
func spentAsWords(spent promptflow.Usage) string {
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
