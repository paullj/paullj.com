package content

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	footnotDefStartRe = regexp.MustCompile(`^\[\^([^\]]+)\]:\s*(.*)`)
	footnoteRefRe     = regexp.MustCompile(`\[\^([^\]]+)\]`)
)

// extractFootnoteDefs parses footnote definitions from markdown and removes them.
// Returns cleaned markdown and a map of label → content.
func extractFootnoteDefs(md string) (string, map[string]string) {
	defs := make(map[string]string)
	lines := strings.Split(md, "\n")

	var out []string
	var curLabel string
	var curLines []string

	flushDef := func() {
		if curLabel != "" {
			defs[curLabel] = strings.TrimSpace(strings.Join(curLines, "\n"))
			curLabel = ""
			curLines = nil
		}
	}

	for _, line := range lines {
		if m := footnotDefStartRe.FindStringSubmatch(line); m != nil {
			flushDef()
			curLabel = m[1]
			if m[2] != "" {
				curLines = []string{m[2]}
			}
			continue
		}

		if curLabel != "" {
			if strings.HasPrefix(line, "    ") || line == "" {
				curLines = append(curLines, strings.TrimPrefix(line, "    "))
				continue
			}
			// Not a continuation line — end current def
			flushDef()
		}

		out = append(out, line)
	}
	flushDef()

	return strings.Join(out, "\n"), defs
}

// extractFootnoteRefs replaces [^label] references with FNREF{label} placeholders.
// Returns modified markdown and ordered list of labels by first appearance.
func extractFootnoteRefs(md string, defs map[string]string) (string, []string) {
	seen := make(map[string]bool)
	var orderedLabels []string

	result := footnoteRefRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := footnoteRefRe.FindStringSubmatch(match)
		label := sub[1]
		if _, ok := defs[label]; !ok {
			return match // undefined ref, leave as-is
		}
		if !seen[label] {
			seen[label] = true
			orderedLabels = append(orderedLabels, label)
		}
		return fmt.Sprintf("FNREF{%s}", label)
	})

	return result, orderedLabels
}

// replaceFootnoteRefs replaces FNREF{label} placeholders with styled [N] markers.
func replaceFootnoteRefs(rendered string, labelOrder []string, darkTheme bool) string {
	labelToNum := make(map[string]int)
	for i, label := range labelOrder {
		labelToNum[label] = i + 1
	}

	accentColor := "33"
	if !darkTheme {
		accentColor = "27"
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(accentColor)).Bold(true)

	fnrefRe := regexp.MustCompile(`FNREF\{([^}]+)\}`)
	return fnrefRe.ReplaceAllStringFunc(rendered, func(match string) string {
		sub := fnrefRe.FindStringSubmatch(match)
		label := sub[1]
		num, ok := labelToNum[label]
		if !ok {
			return match
		}
		return style.Render(fmt.Sprintf("[%d]", num))
	})
}

// renderFootnoteSection renders the footnote section appended at the bottom of a post.
func renderFootnoteSection(defs map[string]string, order []string, width int, style string, darkTheme bool) string {
	if len(order) == 0 {
		return ""
	}

	accentColor := "33"
	if !darkTheme {
		accentColor = "27"
	}
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(accentColor)).Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)

	var out strings.Builder

	// Separator
	out.WriteString("\n" + dimStyle.Render(strings.Repeat("─", 20)) + "\n")

	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	for i, label := range order {
		content, ok := defs[label]
		if !ok || content == "" {
			continue
		}

		rendered, err := RenderMarkdown(content, innerWidth, style)
		if err != nil {
			rendered = content
		}
		rendered = strings.TrimSpace(rendered)

		prefix := numStyle.Render(fmt.Sprintf("%d.", i+1))
		lines := strings.Split(rendered, "\n")
		for j, line := range lines {
			trimmed := stripAllLeadingSpaces(line)
			if j == 0 {
				out.WriteString(prefix + " " + trimmed + "\n")
			} else {
				out.WriteString("   " + trimmed + "\n")
			}
		}
	}

	return out.String()
}
