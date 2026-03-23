package content

import (
	"strings"
	"testing"
)

func TestExtractFootnoteDefs_SingleLine(t *testing.T) {
	md := "Some text\n\n[^1]: This is a footnote\n\nMore text"
	cleaned, defs := extractFootnoteDefs(md)

	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	if defs["1"] != "This is a footnote" {
		t.Errorf("expected 'This is a footnote', got %q", defs["1"])
	}
	if strings.Contains(cleaned, "[^1]:") {
		t.Error("definition should be stripped from output")
	}
	if !strings.Contains(cleaned, "Some text") || !strings.Contains(cleaned, "More text") {
		t.Error("non-def content should be preserved")
	}
}

func TestExtractFootnoteDefs_MultiLine(t *testing.T) {
	md := "[^1]:\n    First line\n    https://example.com\n\nMore text"
	cleaned, defs := extractFootnoteDefs(md)

	if defs["1"] != "First line\nhttps://example.com" {
		t.Errorf("unexpected multiline def: %q", defs["1"])
	}
	if strings.Contains(cleaned, "First line") {
		t.Error("multiline def should be stripped")
	}
	if !strings.Contains(cleaned, "More text") {
		t.Error("non-def content should be preserved")
	}
}

func TestExtractFootnoteDefs_NamedLabels(t *testing.T) {
	md := "[^expensive]: This costs too much\n[^cheap]: This is affordable"
	_, defs := extractFootnoteDefs(md)

	if len(defs) != 2 {
		t.Fatalf("expected 2 defs, got %d", len(defs))
	}
	if defs["expensive"] != "This costs too much" {
		t.Errorf("unexpected: %q", defs["expensive"])
	}
	if defs["cheap"] != "This is affordable" {
		t.Errorf("unexpected: %q", defs["cheap"])
	}
}

func TestExtractFootnoteRefs(t *testing.T) {
	defs := map[string]string{"1": "note one", "2": "note two"}
	md := "Text[^1] and more[^2] and again[^1]"
	result, order := extractFootnoteRefs(md, defs)

	if len(order) != 2 {
		t.Fatalf("expected 2 ordered labels, got %d", len(order))
	}
	if order[0] != "1" || order[1] != "2" {
		t.Errorf("unexpected order: %v", order)
	}
	if !strings.Contains(result, "FNREF{1}") || !strings.Contains(result, "FNREF{2}") {
		t.Errorf("expected placeholders, got %q", result)
	}
	if strings.Contains(result, "[^1]") || strings.Contains(result, "[^2]") {
		t.Error("original refs should be replaced")
	}
}

func TestExtractFootnoteRefs_UndefinedRef(t *testing.T) {
	defs := map[string]string{"1": "note one"}
	md := "Text[^1] and[^undefined]"
	result, order := extractFootnoteRefs(md, defs)

	if len(order) != 1 {
		t.Fatalf("expected 1 ordered label, got %d", len(order))
	}
	if !strings.Contains(result, "[^undefined]") {
		t.Error("undefined ref should be left as-is")
	}
}

func TestExtractFootnoteRefs_AfterDefsStripped(t *testing.T) {
	// In the real pipeline, defs are already stripped before refs are replaced
	md := "Some text\n\n[^1]: This is a def\n\nText[^1]"
	cleaned, defs := extractFootnoteDefs(md)
	result, order := extractFootnoteRefs(cleaned, defs)

	if len(order) != 1 {
		t.Fatalf("expected 1 ordered label, got %d", len(order))
	}
	if !strings.Contains(result, "FNREF{1}") {
		t.Error("ref should be replaced with placeholder")
	}
}

func TestReplaceFootnoteRefs(t *testing.T) {
	rendered := "text FNREF{1} more FNREF{2} again FNREF{1}"
	result := replaceFootnoteRefs(rendered, []string{"1", "2"}, true)

	if strings.Contains(result, "FNREF") {
		t.Error("placeholders should be replaced")
	}
	// Should contain [1] and [2] (with ANSI styling)
	if !strings.Contains(result, "1") || !strings.Contains(result, "2") {
		t.Error("should contain footnote numbers")
	}
}

func TestRenderFootnoteSection_Empty(t *testing.T) {
	result := renderFootnoteSection(nil, nil, 80, "dark", true)
	if result != "" {
		t.Error("empty order should return empty string")
	}
}

func TestRenderFootnoteSection_WithContent(t *testing.T) {
	defs := map[string]string{"a": "First note", "b": "Second note"}
	order := []string{"a", "b"}
	result := renderFootnoteSection(defs, order, 80, "dark", true)

	if !strings.Contains(result, "─") {
		t.Error("should contain separator")
	}
	// Glamour strips ANSI but content words should still be present
	stripped := ansiRe.ReplaceAllString(result, "")
	if !strings.Contains(stripped, "First note") || !strings.Contains(stripped, "Second note") {
		t.Errorf("should contain footnote content, got:\n%s", result)
	}
}
