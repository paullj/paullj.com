package tui

import (
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type splashState struct {
	charIndex int
	done      bool
}

type (
	splashTickMsg struct{}
	splashDoneMsg struct{}
)

func (m Model) splashCharDelay() time.Duration {
	base := m.cfg.SSH.Splash.CharDelay.Duration
	jitter := m.cfg.SSH.Splash.CharJitter.Duration
	if jitter > 0 {
		base += time.Duration(rand.Int63n(int64(jitter)))
	}
	return base
}

func (m Model) splashTick() tea.Cmd {
	d := m.splashCharDelay()
	return tea.Tick(d, func(time.Time) tea.Msg {
		return splashTickMsg{}
	})
}

func (m Model) splashHold() tea.Cmd {
	return tea.Tick(m.cfg.SSH.Splash.HoldTime.Duration, func(time.Time) tea.Msg {
		return splashDoneMsg{}
	})
}

func (m Model) updateSplash(msg tea.Msg) (tea.Model, tea.Cmd) {
	text := m.cfg.SSH.Splash.Text
	switch msg.(type) {
	case splashTickMsg:
		if m.splash.charIndex < len(text) {
			m.splash.charIndex++
			if m.splash.charIndex < len(text) {
				return m, m.splashTick()
			}
			return m, m.splashHold()
		}
	case splashDoneMsg:
		m.splash.done = true
		return m, nil
	}
	return m, nil
}

func (m Model) viewSplash() string {
	text := m.cfg.SSH.Splash.Text
	typed := text[:m.splash.charIndex]
	remaining := len(text) - len(typed)

	accent := m.accentColor()
	caretStyle := lipgloss.NewStyle().Foreground(accent)

	// Typed text + colored caret + spaces for remaining chars to keep left-aligned
	content := typed + caretStyle.Render("█")
	if remaining > 0 {
		content += strings.Repeat(" ", remaining)
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
