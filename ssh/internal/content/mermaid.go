package content

import (
	"fmt"
	"regexp"
	"strings"

	mermaid "github.com/AlexanderGrooff/mermaid-ascii/cmd"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"

	"charm.land/lipgloss/v2"
)

var mermaidRe = regexp.MustCompile("(?ms)```mermaid\\s*\\n(.+?)\\n```")

type mermaidRef struct {
	marker string
	source string
}

// MermaidDiagram is returned to the TUI layer for diagrams that overflow content width.
type MermaidDiagram struct {
	Index    int
	Rendered string
}

func extractMermaid(md string) (string, []mermaidRef) {
	var refs []mermaidRef
	result := mermaidRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := mermaidRe.FindStringSubmatch(match)
		marker := fmt.Sprintf("MERMAIDPLACEHOLDER%d", len(refs))
		refs = append(refs, mermaidRef{marker: marker, source: sub[1]})
		return marker
	})
	return result, refs
}

func renderMermaidDiagram(source string) (string, error) {
	config := diagram.DefaultConfig()
	return mermaid.RenderDiagram(source, config)
}

func replaceMermaid(rendered string, refs []mermaidRef, width int, darkTheme bool) (string, []MermaidDiagram) {
	var overflows []MermaidDiagram

	for i, ref := range refs {
		output, err := renderMermaidDiagram(ref.source)
		if err != nil {
			// Fallback: leave as code block text
			placeholder := regexp.MustCompile(`[^\n]*` + regexp.QuoteMeta(ref.marker) + `[^\n]*`)
			rendered = placeholder.ReplaceAllLiteralString(rendered, ref.source)
			continue
		}

		output = strings.TrimRight(output, "\n")

		// Measure max line width
		maxW := 0
		for _, line := range strings.Split(output, "\n") {
			w := lipgloss.Width(line)
			if w > maxW {
				maxW = w
			}
		}

		placeholder := regexp.MustCompile(`[^\n]*` + regexp.QuoteMeta(ref.marker) + `[^\n]*`)

		if maxW <= width {
			// Inline the diagram
			rendered = placeholder.ReplaceAllLiteralString(rendered, output)
		} else {
			// Overflow: show placeholder, add to overflow list
			dim := lipgloss.Color("241")
			accent := lipgloss.Color("170")
			if !darkTheme {
				dim = lipgloss.Color("245")
				accent = lipgloss.Color("63")
			}
			borderStyle := lipgloss.NewStyle().Foreground(dim)
			textStyle := lipgloss.NewStyle().Foreground(accent)

			label := textStyle.Render("▸ Diagram — press enter to expand")
			labelW := lipgloss.Width(label) + 4
			box := borderStyle.Render("┌"+strings.Repeat("─", labelW)+"┐") + "\n" +
				borderStyle.Render("│") + "  " + label + "  " + borderStyle.Render("│") + "\n" +
				borderStyle.Render("└"+strings.Repeat("─", labelW)+"┘")

			rendered = placeholder.ReplaceAllLiteralString(rendered, box)
			overflows = append(overflows, MermaidDiagram{Index: i, Rendered: output})
		}
	}

	return rendered, overflows
}
