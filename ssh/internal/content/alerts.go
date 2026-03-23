package content

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

var alertRe = regexp.MustCompile(`(?m)^> \[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*\n((?:>[ ]?[^\n]*\n?)*)`)

// ansiRe matches ANSI escape sequences.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type alertRef struct {
	marker    string
	alertType string
	inner     string
}

func alertColor(alertType string, dark bool) string {
	colors := map[string][2]string{
		"NOTE":      {"33", "27"},
		"TIP":       {"42", "28"},
		"IMPORTANT": {"135", "91"},
		"WARNING":   {"214", "208"},
		"CAUTION":   {"196", "160"},
	}
	c := colors[alertType]
	if dark {
		return c[0]
	}
	return c[1]
}

func extractAlerts(md string) (string, []alertRef) {
	var refs []alertRef
	result := alertRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := alertRe.FindStringSubmatch(match)
		marker := fmt.Sprintf("ALERTPLACEHOLDER%d", len(refs))
		// Strip "> " prefix from inner lines
		var lines []string
		for _, line := range strings.Split(sub[2], "\n") {
			line = strings.TrimPrefix(line, "> ")
			line = strings.TrimPrefix(line, ">")
			lines = append(lines, line)
		}
		inner := strings.TrimSpace(strings.Join(lines, "\n"))
		refs = append(refs, alertRef{marker: marker, alertType: sub[1], inner: inner})
		return marker
	})
	return result, refs
}

// stripAllLeadingSpaces removes all leading visible spaces from a line,
// skipping over any ANSI escape sequences at the start.
func stripAllLeadingSpaces(line string) string {
	i := 0
	for i < len(line) {
		if loc := ansiRe.FindStringIndex(line[i:]); loc != nil && loc[0] == 0 {
			i += loc[1]
			continue
		}
		if line[i] == ' ' {
			i++
		} else {
			break
		}
	}
	return line[i:]
}

func replaceAlerts(rendered string, refs []alertRef, width int, style string, darkTheme bool) string {
	for _, ref := range refs {
		color := alertColor(ref.alertType, darkTheme)
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true)
		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		border := borderStyle.Render("┃")

		var body string
		if ref.inner != "" {
			innerWidth := width - 4
			if innerWidth < 20 {
				innerWidth = 20
			}
			innerRendered, err := RenderMarkdown(ref.inner, innerWidth, style)
			if err != nil {
				innerRendered = ref.inner
			}
			body = strings.TrimSpace(innerRendered)
		}

		var out strings.Builder
		out.WriteString(" " + border + " " + labelStyle.Render(ref.alertType) + "\n")
		if body != "" {
			for _, line := range strings.Split(body, "\n") {
				trimmed := stripAllLeadingSpaces(line)
				out.WriteString(" " + border + " " + trimmed + "\n")
			}
		}

		placeholder := regexp.MustCompile(`[^\n]*` + regexp.QuoteMeta(ref.marker) + `[^\n]*`)
		rendered = placeholder.ReplaceAllLiteralString(rendered, strings.TrimRight(out.String(), "\n")+"\n")
	}
	return rendered
}
