package overlay

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/wazum/herdr-polyglot/internal/translation"
	"github.com/wazum/herdr-polyglot/internal/vimarea"
)

// Herdr paints the terminal palette from the active theme but does not expose
// the theme itself, so the overlay only names palette slots and lets whatever
// theme is running decide how they look.
var (
	accent = lipgloss.Color("5")
	muted  = lipgloss.Color("8")
	danger = lipgloss.Color("1")
	frame  = lipgloss.Color("8")

	accentStyle      = lipgloss.NewStyle().Foreground(accent)
	textStyle        = lipgloss.NewStyle()
	placeholderStyle = lipgloss.NewStyle().Foreground(muted)
	cursorStyle      = lipgloss.NewStyle().Foreground(accent)
	titleStyle       = lipgloss.NewStyle().Foreground(accent).Bold(true)
	badgeStyle       = lipgloss.NewStyle().Foreground(muted)
	hintStyle        = lipgloss.NewStyle().Foreground(muted)
	keyStyle         = lipgloss.NewStyle().Foreground(accent)
	dangerStyle      = lipgloss.NewStyle().Foreground(danger)
	modeStyle        = lipgloss.NewStyle().Foreground(accent).Bold(true)

	paneStyle = lipgloss.NewStyle()

	draftBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(frame).
			Padding(0, 1)

	englishBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1).
			Height(englishRows - 2)
)

func (m Model) View() string {
	draft := draftBoxStyle.Width(m.width).Render(m.draft.View())
	line := lipgloss.Width(draft)

	parts := []string{m.header(line), draft}
	if m.showsEnglish() {
		parts = append(parts, m.englishPane())
	}
	parts = append(parts, m.footer(line))

	// Herdr already draws a frame around the popup; a second one inside it only
	// takes room from the draft.
	return m.fillPane(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m Model) fillPane(content string) string {
	if m.pane.Width <= 0 || m.pane.Height <= 0 {
		return content
	}
	return paneStyle.Width(m.pane.Width).Height(m.pane.Height).Render(content)
}

func (m Model) englishPane() string {
	body := placeholderStyle.Render("…")
	switch {
	case m.previewError != nil:
		body = dangerStyle.Render("✗ " + m.previewError.Error())
	case m.preview != "":
		body = textStyle.Render(m.preview)
		if !m.previewIsCurrent() {
			body = placeholderStyle.Render(m.preview)
		}
	}
	return englishBoxStyle.Width(m.width).Render(body)
}

func (m Model) header(line int) string {
	destination := "send"
	if m.options.Review {
		destination = "review"
	}
	if m.options.Live {
		live := "live"
		if m.options.Pulse {
			live = m.pulseGlyph() + " live"
		}
		destination = live + " · " + destination
	}
	parts := []string{m.options.Service, "→", m.options.Language, "·", destination}
	if m.resumed {
		parts = append(parts, "·", "resumed")
	}
	if m.spentKnown {
		parts = append(parts, "·", spentAsWords(m.spent))
	}
	badge := badgeStyle.Render(strings.Join(parts, " "))

	return spread(titleStyle.Render("✳ polyglot"), badge, line)
}

func (m Model) footer(line int) string {
	mode := ""
	if m.draft.Modal() {
		mode = modeStyle.Render(m.draft.Mode().String()) + " "
	}

	switch {
	case m.failure != nil:
		return spread(mode+dangerStyle.Render("✗ "+m.failure.Error()), hintStyle.Render("ctrl+c close"), line)
	case m.draftIsTooLong():
		return spread(
			mode+dangerStyle.Render(fmt.Sprintf("⚠ %d characters", len([]rune(m.draft.Value()))))+
				hintStyle.Render(" — this box is for prompts you write, not files you paste"),
			hintStyle.Render("ctrl+u discard"), line)
	case m.stage == confirming:
		return spread(
			keyStyle.Render("ctrl+d")+hintStyle.Render(" send this")+
				hintStyle.Render(" · ")+keyStyle.Render("esc")+hintStyle.Render(" keep writing"),
			badgeStyle.Render("read it first"), line)
	case m.stage == translating:
		return spread(mode+m.spinner.View()+accentStyle.Render(" translating …"), "", line)
	default:
		return spread(mode+" "+m.keyHints(), "", line)
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
		hints = append(hints, keyStyle.Render(hint[0])+hintStyle.Render(" "+hint[1]))
	}
	return strings.Join(hints, hintStyle.Render(" · "))
}

// spread pins left to the start of the line and right to its end.
func spread(left, right string, line int) string {
	gap := line - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
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
