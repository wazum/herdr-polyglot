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
	if m.reading {
		return m.readingView()
	}

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

// While reading, the translation has every row the draft and its box were using.
func (m Model) readingView() string {
	rows := m.readingRows()
	total := m.readingTotal()

	text, style := m.english()
	if m.preview == "" {
		text = "nothing translated yet"
	}
	shown := style.Render(rowsFrom(text, m.contentWidth(), m.readingFrom, rows))

	box := m.styles.englishBox.Width(m.width).Height(rows).
		Render(m.scrolled(shown, m.readingFrom, rows, total))
	line := lipgloss.Width(box)

	return strings.Join([]string{m.header(line), box, m.readingFooter(line - 1)}, "\n")
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

// Beside the draft there is room to glance at the translation, not to read it, so
// the panel shows where it starts and marks that it goes on. Tab is what reads it.
// The text is fitted before it is styled: escape sequences in the middle of it
// would decide where the lines break.
func (m Model) englishPane() string {
	text, style := m.english()

	rows := englishRows - 2
	shown := rowsFrom(text, m.contentWidth(), 0, rows)
	if m.translationIsCut() {
		shown = cutTo(shown, m.contentWidth()-2) + " …"
	}
	return m.styles.englishBox.Width(m.width).Render(style.Render(shown))
}

func (m Model) translationIsCut() bool {
	return m.preview != "" && rowsOf(m.preview, m.contentWidth()) > englishRows-2
}

// english is the translation as it stands, and how it should read: dimmed while it
// belongs to an older draft, in the danger colour when the service said no.
func (m Model) english() (string, lipgloss.Style) {
	switch {
	case m.previewError != nil:
		return "✗ " + m.previewError.Error(), m.styles.danger
	case m.preview == "":
		return "…", m.styles.placeholder
	case !m.previewIsCurrent():
		return m.preview, m.styles.placeholder
	default:
		return m.preview, m.styles.text
	}
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
	shown := [][2]string{{"alt+enter", m.destination()}}
	// Only worth naming while there is more translation than the panel shows.
	if m.translationIsCut() {
		shown = append(shown, [2]string{"tab", "read it"})
	}
	shown = append(shown,
		[2]string{"ctrl+r", "→ " + m.otherDestination()},
		[2]string{"ctrl+l", "live"})

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

// The keys are few enough here to name them all, and the last one is the way out.
func (m Model) readingFooter(inner int) string {
	hints := []string{
		m.styles.key.Render("↑↓") + m.styles.hint.Render(" read"),
		m.styles.key.Render("tab") + m.styles.hint.Render(" → write"),
		m.styles.key.Render("alt+enter") + m.styles.hint.Render(" "+m.destination()),
		m.styles.key.Render("esc") + m.styles.hint.Render(" back"),
	}
	return spread(" "+strings.Join(hints, m.styles.hint.Render(" · ")),
		m.styles.badge.Render(m.howFar()), inner)
}

// howFar says where in the translation the reader is, since the bar shows it only
// roughly.
func (m Model) howFar() string {
	rows, total := m.readingRows(), m.readingTotal()
	if total <= rows {
		return ""
	}
	return fmt.Sprintf("%d%%", min((m.readingFrom+rows)*100/total, 100))
}

// rowsFrom wraps text and keeps the rows from one onwards.
func rowsFrom(text string, width, from, rows int) string {
	if width < 1 || rows < 1 {
		return text
	}

	lines := strings.Split(wrapped(text, width), "\n")
	from = min(max(from, 0), max(len(lines)-1, 0))
	return strings.Join(lines[from:min(from+rows, len(lines))], "\n")
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
