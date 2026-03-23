package content

import (
	"strings"
	"testing"
)

func TestExtractMermaid_Single(t *testing.T) {
	md := "Some text.\n\n```mermaid\ngraph LR\n    A --> B\n```\n\nMore text."

	result, refs := extractMermaid(md)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if !strings.Contains(result, "MERMAIDPLACEHOLDER0") {
		t.Error("result should contain placeholder")
	}
	if strings.Contains(result, "```mermaid") {
		t.Error("result should not contain mermaid fence")
	}
	if !strings.Contains(refs[0].source, "graph LR") {
		t.Error("source should contain graph LR")
	}
	if !strings.Contains(result, "Some text.") {
		t.Error("surrounding text should be preserved")
	}
}

func TestExtractMermaid_None(t *testing.T) {
	md := "No mermaid here.\n\n```go\nfmt.Println()\n```"

	result, refs := extractMermaid(md)

	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(refs))
	}
	if result != md {
		t.Error("should be unchanged")
	}
}

func TestExtractMermaid_Multiple(t *testing.T) {
	md := "```mermaid\ngraph LR\n    A --> B\n```\n\ntext\n\n```mermaid\ngraph TD\n    C --> D\n```"

	result, refs := extractMermaid(md)

	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if !strings.Contains(result, "MERMAIDPLACEHOLDER0") || !strings.Contains(result, "MERMAIDPLACEHOLDER1") {
		t.Error("should contain both placeholders")
	}
}

func TestRenderMermaidDiagram_SimpleGraph(t *testing.T) {
	source := "graph LR\n    A --> B"

	output, err := renderMermaidDiagram(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output == "" {
		t.Error("output should not be empty")
	}
}

func TestReplaceMermaid_InlineFit(t *testing.T) {
	rendered := "before\nMERMAIDPLACEHOLDER0\nafter"
	refs := []mermaidRef{{marker: "MERMAIDPLACEHOLDER0", source: "graph LR\n    A --> B"}}

	result, overflows := replaceMermaid(rendered, refs, 200, true)

	if len(overflows) != 0 {
		t.Error("wide width should produce no overflows")
	}
	if strings.Contains(result, "MERMAIDPLACEHOLDER0") {
		t.Error("placeholder should be replaced")
	}
	if !strings.Contains(result, "before") || !strings.Contains(result, "after") {
		t.Error("surrounding text should be preserved")
	}
}

func TestReplaceMermaid_Overflow(t *testing.T) {
	rendered := "before\nMERMAIDPLACEHOLDER0\nafter"
	refs := []mermaidRef{{marker: "MERMAIDPLACEHOLDER0", source: "graph LR\n    A --> B"}}

	result, overflows := replaceMermaid(rendered, refs, 5, true) // very narrow

	if len(overflows) != 1 {
		t.Fatalf("expected 1 overflow, got %d", len(overflows))
	}
	if !strings.Contains(result, "press enter to expand") {
		t.Error("should contain overflow placeholder text")
	}
}

func TestReplaceMermaid_RenderError(t *testing.T) {
	rendered := "before\nMERMAIDPLACEHOLDER0\nafter"
	refs := []mermaidRef{{marker: "MERMAIDPLACEHOLDER0", source: "not valid mermaid at all!!!"}}

	result, overflows := replaceMermaid(rendered, refs, 200, true)

	if len(overflows) != 0 {
		t.Error("error should produce no overflows")
	}
	if strings.Contains(result, "MERMAIDPLACEHOLDER0") {
		t.Error("placeholder should be replaced with fallback")
	}
}
