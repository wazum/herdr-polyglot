package overlay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/wazum/herdr-polyglot/internal/translation"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

func (m Model) View() string {
	draft := m.styles.draftBox.Width(m.width).Render(m.draft.View())
	line := lipgloss.Width(draft)

	parts := []string{m.header(), draft}
	if m.showsEnglish() {
		parts = append(parts, m.englishPane())
	}
	parts = append(parts, m.footer(line))

	// Herdr already draws a frame around the popup; a second one inside it only
	// takes room from the draft.
	return m.fillPane(strings.Join(parts, "\n"))
}

// With herdr's own colour in hand every cell can be painted, which leaves the
// popup one shade throughout. Without it, cells are left untouched instead:
// a space in the wrong shade is worse than one herdr painted itself.
func (m Model) fillPane(content string) string {
	if !m.styles.known || m.pane.Width <= 0 || m.pane.Height <= 0 {
		return content
	}
	return m.styles.pane.Width(m.pane.Width).Height(m.pane.Height).Render(content)
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
// badge saying what will happen. It starts one column in, under that name.
func (m Model) header() string {
	return m.styles.text.Render(" ") + m.badge()
}

// The badge is joined from rendered pieces: the pulse carries its own colour,
// which a single Render around everything would cut short.
func (m Model) badge() string {
	pieces := []string{m.styles.badge.Render(m.options.Service + " → " + m.options.Language)}

	if m.options.Live {
		if m.options.Pulse {
			pieces = append(pieces,
				m.styles.badge.Render("·"), m.pulseGlyph(), m.styles.badge.Render("live"))
		} else {
			pieces = append(pieces, m.styles.badge.Render("· live"))
		}
	}

	pieces = append(pieces, m.styles.badge.Render("· "+m.destination()))
	if m.resumed {
		pieces = append(pieces, m.styles.badge.Render("· resumed"))
	}
	if m.spentKnown {
		pieces = append(pieces, m.styles.badge.Render("· "+spentAsWords(m.spent)))
	}
	return strings.Join(pieces, m.styles.badge.Render(" "))
}

func (m Model) destination() string {
	if m.options.Review {
		return "review"
	}
	return "send"
}

func (m Model) footer(line int) string {
	mode := ""
	if m.draft.Modal() {
		mode = m.styles.mode.Render(m.draft.Mode().String()) + m.styles.text.Render(" ")
	}

	switch {
	case m.failure != nil:
		return m.spread(mode+m.styles.danger.Render("✗ "+m.failure.Error()),
			m.styles.hint.Render("ctrl+c close"), line)
	case m.draftIsTooLong():
		return m.spread(
			mode+m.styles.danger.Render(fmt.Sprintf("⚠ %d characters", len([]rune(m.draft.Value()))))+
				m.styles.hint.Render(" — this box is for prompts you write, not files you paste"),
			m.styles.hint.Render("ctrl+u discard"), line)
	case m.stage == confirming:
		return m.spread(
			m.styles.key.Render("ctrl+d")+m.styles.hint.Render(" send this")+
				m.styles.hint.Render(" · ")+m.styles.key.Render("esc")+
				m.styles.hint.Render(" keep writing"),
			m.styles.badge.Render("read it first"), line)
	case m.stage == translating:
		return m.spread(mode+m.spinner.View()+m.styles.accent.Render(" translating …"), "", line)
	default:
		return m.spread(mode+m.styles.text.Render(" ")+m.keyHints(), "", line)
	}
}

func (m Model) keyHints() string {
	shown := [][2]string{{"ctrl+d", "send"}, {"enter", "newline"}, {"esc", "close"}}
	switch {
	case m.draft.Modal() && m.draft.Mode() == vimarea.Normal:
		shown = [][2]string{{"ctrl+d", "send"}, {"i", "insert"}, {"q", "close"}}
	case m.draft.Modal():
		shown = [][2]string{{"ctrl+d", "send"}, {"esc", "normal"}, {"enter", "newline"}}
	}
	if m.resumed {
		shown = append(shown, [2]string{"ctrl+u", "discard"})
	}

	hints := make([]string, 0, len(shown))
	for _, hint := range shown {
		hints = append(hints, m.styles.key.Render(hint[0])+m.styles.hint.Render(" "+hint[1]))
	}
	return strings.Join(hints, m.styles.hint.Render(" · "))
}

// spread pins left to the start of the line and right to its end. With nothing
// on the right the line ends there unless the popup's own colour is known, in
// which case filling it out keeps the shade even.
func (m Model) spread(left, right string, line int) string {
	if right == "" && !m.styles.known {
		return left
	}

	gap := line - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		if right == "" {
			return left
		}
		return left + m.styles.text.Render(" ") + right
	}
	return left + m.styles.text.Render(strings.Repeat(" ", gap)) + right
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
