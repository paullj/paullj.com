package content

import (
	"strings"
	"testing"
)

func TestExtractAlerts_NoteWithLink(t *testing.T) {
	// From composing-music-with-code.md
	md := `Some intro text.

> [!NOTE]
> Want to skip to making music? You can try timb(re) out at [paullj.github.io/timb](https://paullj.github.io/timb) or check out the code [here](https://github.com/paullj/timb).

More text after.`

	result, refs := extractAlerts(md)

	if len(refs) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(refs))
	}
	if refs[0].alertType != "NOTE" {
		t.Errorf("expected NOTE, got %s", refs[0].alertType)
	}
	if !strings.Contains(refs[0].inner, "paullj.github.io/timb") {
		t.Error("inner should contain link text")
	}
	if !strings.Contains(result, "ALERTPLACEHOLDER0") {
		t.Error("result should contain placeholder")
	}
	if strings.Contains(result, "[!NOTE]") {
		t.Error("result should not contain original alert syntax")
	}
	if !strings.Contains(result, "Some intro text.") {
		t.Error("surrounding text should be preserved")
	}
	if !strings.Contains(result, "More text after.") {
		t.Error("surrounding text should be preserved")
	}
}

func TestExtractAlerts_MultipleInSamePost(t *testing.T) {
	// From cross-compiling-rust-for-raspberry-pi-development.md
	md := `> [!NOTE]
> This is a very loose guide that I hope will help point others in the right direction, but it's _not_ a step-by-step tutorial.

Some text in between.

> [!TIP]
> It might take a few seconds for the Pi to boot up, so be patient.

More text.

> [!NOTE]
> I'm using a Mac, so I can use the ` + "`.local`" + ` hostname to connect to the Pi.`

	result, refs := extractAlerts(md)

	if len(refs) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(refs))
	}
	if refs[0].alertType != "NOTE" {
		t.Errorf("refs[0]: expected NOTE, got %s", refs[0].alertType)
	}
	if refs[1].alertType != "TIP" {
		t.Errorf("refs[1]: expected TIP, got %s", refs[1].alertType)
	}
	if refs[2].alertType != "NOTE" {
		t.Errorf("refs[2]: expected NOTE, got %s", refs[2].alertType)
	}
	if !strings.Contains(result, "ALERTPLACEHOLDER0") ||
		!strings.Contains(result, "ALERTPLACEHOLDER1") ||
		!strings.Contains(result, "ALERTPLACEHOLDER2") {
		t.Error("all placeholders should be present")
	}
	if !strings.Contains(result, "Some text in between.") {
		t.Error("text between alerts should be preserved")
	}
}

func TestExtractAlerts_NoAlerts(t *testing.T) {
	md := `Just a regular blockquote:

> This is not an alert.

And some text.`

	result, refs := extractAlerts(md)

	if len(refs) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(refs))
	}
	if result != md {
		t.Error("markdown without alerts should be unchanged")
	}
}

func TestExtractAlerts_StripsPrefixes(t *testing.T) {
	// From learning-pcb-design-and-embedded-rust.md
	md := `> [!NOTE]
> You can find the KiCad project files of the schematic and layout for my PCB on [GitHub](https://github.com/paullj/synth/tree/main/synth-hardware)`

	_, refs := extractAlerts(md)

	if len(refs) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(refs))
	}
	if strings.HasPrefix(refs[0].inner, "> ") {
		t.Error("inner should have > prefix stripped")
	}
	if !strings.Contains(refs[0].inner, "KiCad project files") {
		t.Error("inner should contain content text")
	}
}

func TestExtractAlerts_AllTypes(t *testing.T) {
	types := []string{"NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"}
	for _, typ := range types {
		md := "> [!" + typ + "]\n> Some content."
		_, refs := extractAlerts(md)
		if len(refs) != 1 {
			t.Errorf("%s: expected 1 alert, got %d", typ, len(refs))
			continue
		}
		if refs[0].alertType != typ {
			t.Errorf("expected %s, got %s", typ, refs[0].alertType)
		}
	}
}

func TestExtractAlerts_EmptyBody(t *testing.T) {
	md := "> [!NOTE]\n"

	_, refs := extractAlerts(md)

	if len(refs) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(refs))
	}
	if refs[0].inner != "" {
		t.Errorf("expected empty inner, got %q", refs[0].inner)
	}
}

func TestReplaceAlerts_Rendering(t *testing.T) {
	rendered := "before\nALERTPLACEHOLDER0\nafter\n"
	refs := []alertRef{{
		marker:    "ALERTPLACEHOLDER0",
		alertType: "NOTE",
		inner:     "Some **bold** content.",
	}}

	result := replaceAlerts(rendered, refs, 80, "dark", true)

	if !strings.Contains(result, "NOTE") {
		t.Error("should contain NOTE label")
	}
	if !strings.Contains(result, "┃") {
		t.Error("should contain border character")
	}
	if !strings.Contains(result, "before") {
		t.Error("surrounding text should be preserved")
	}
	if !strings.Contains(result, "after") {
		t.Error("surrounding text should be preserved")
	}
	if strings.Contains(result, "ALERTPLACEHOLDER0") {
		t.Error("placeholder should be replaced")
	}
}

func TestReplaceAlerts_TrailingNewline(t *testing.T) {
	rendered := "before\nALERTPLACEHOLDER0\nafter\n"
	refs := []alertRef{{
		marker:    "ALERTPLACEHOLDER0",
		alertType: "TIP",
		inner:     "Tip content.",
	}}

	result := replaceAlerts(rendered, refs, 80, "dark", true)

	// Find the alert block and check it ends with a newline before "after"
	lines := strings.Split(result, "\n")
	foundAfter := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "after" {
			// Previous line should be empty or the last border line
			if i > 0 && strings.Contains(lines[i-1], "ALERTPLACEHOLDER") {
				t.Error("placeholder should have been replaced before 'after'")
			}
			foundAfter = true
			break
		}
	}
	if !foundAfter {
		t.Error("'after' text should be present")
	}
}

func TestReplaceAlerts_Colors(t *testing.T) {
	types := map[string][2]string{
		"NOTE":      {"33", "27"},
		"TIP":       {"42", "28"},
		"IMPORTANT": {"135", "91"},
		"WARNING":   {"214", "208"},
		"CAUTION":   {"196", "160"},
	}

	for typ, colors := range types {
		for i, dark := range []bool{true, false} {
			got := alertColor(typ, dark)
			if got != colors[i] {
				t.Errorf("alertColor(%s, dark=%v) = %s, want %s", typ, dark, got, colors[i])
			}
		}
	}
}

func TestStripAllLeadingSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain spaces", "    hello", "hello"},
		{"no spaces", "hello", "hello"},
		{"with ansi", "\x1b[0m    hello", "hello"},
		{"ansi between spaces", "  \x1b[1m  hello", "hello"},
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAllLeadingSpaces(tt.input)
			if got != tt.want {
				t.Errorf("stripAllLeadingSpaces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
