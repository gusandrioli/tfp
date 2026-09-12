package render

import (
	"testing"

	"github.com/gusandrioli/tfp/internal/planmodel"
)

func TestDiffSymbolAndRHS(t *testing.T) {
	cases := []struct {
		name       string
		diff       planmodel.AttributeDiff
		wantSymbol string
		wantRHS    string
	}{
		{
			name:       "plain update",
			diff:       planmodel.AttributeDiff{Before: "9.2.2", After: "9.2.4"},
			wantSymbol: "~",
			wantRHS:    `"9.2.2" -> "9.2.4"`,
		},
		{
			name:       "added",
			diff:       planmodel.AttributeDiff{Before: nil, After: "us-east-1"},
			wantSymbol: "+",
			wantRHS:    `"us-east-1"`,
		},
		{
			name:       "removed",
			diff:       planmodel.AttributeDiff{Before: "platform", After: nil},
			wantSymbol: "-",
			wantRHS:    `"platform"`,
		},
		{
			// This is the replace-triggered "id" case: the resource is
			// being recreated, so the old id is known but the new one
			// isn't — After is nil in the plan JSON, but that must not
			// be confused with the value being removed.
			name:       "unknown with a known before value",
			diff:       planmodel.AttributeDiff{Before: "abc123", After: nil, Unknown: true},
			wantSymbol: "~",
			wantRHS:    `"abc123" -> (known after apply)`,
		},
		{
			name:       "unknown on create (no prior value)",
			diff:       planmodel.AttributeDiff{Before: nil, After: nil, Unknown: true},
			wantSymbol: "+",
			wantRHS:    "(known after apply)",
		},
		{
			name:       "sensitive",
			diff:       planmodel.AttributeDiff{Before: "old-secret", After: "new-secret", Sensitive: true},
			wantSymbol: "~",
			wantRHS:    "(sensitive value) -> (sensitive value)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			symbol, rhs := diffSymbolAndRHS(c.diff)
			if symbol != c.wantSymbol || rhs != c.wantRHS {
				t.Errorf("diffSymbolAndRHS(%+v) = (%q, %q), want (%q, %q)", c.diff, symbol, rhs, c.wantSymbol, c.wantRHS)
			}
		})
	}
}

func TestFormatAttributeLine(t *testing.T) {
	changed := planmodel.AttributeDiff{
		Path: planmodel.AttributePath{{Key: "content"}}, Before: "v1", After: "v2", Changed: true,
	}
	if got, want := FormatAttributeLine(changed), `~ content = "v1" -> "v2"`; got != want {
		t.Errorf("FormatAttributeLine(changed) = %q, want %q", got, want)
	}

	unchanged := planmodel.AttributeDiff{
		Path: planmodel.AttributePath{{Key: "name"}}, Before: "quoter", After: "quoter", Changed: false,
	}
	if got, want := FormatAttributeLine(unchanged), `  name = "quoter"`; got != want {
		t.Errorf("FormatAttributeLine(unchanged) = %q, want %q", got, want)
	}

	sensitiveUnchanged := planmodel.AttributeDiff{
		Path: planmodel.AttributePath{{Key: "token"}}, Before: "secret", After: "secret", Changed: false, Sensitive: true,
	}
	if got, want := FormatAttributeLine(sensitiveUnchanged), `  token = (sensitive value)`; got != want {
		t.Errorf("FormatAttributeLine(sensitiveUnchanged) = %q, want %q — an unchanged sensitive value must still be redacted", got, want)
	}
}

func TestFormatDiff_ForcesReplacement(t *testing.T) {
	plain := planmodel.AttributeDiff{Path: planmodel.AttributePath{{Key: "content"}}, Before: "v1", After: "v2"}
	if got, want := FormatDiff(plain), `~ content = "v1" -> "v2"`; got != want {
		t.Errorf("FormatDiff(%+v) = %q, want %q", plain, got, want)
	}

	forcing := plain
	forcing.ForcesReplacement = true
	if got, want := FormatDiff(forcing), `~ content = "v1" -> "v2" # forces replacement`; got != want {
		t.Errorf("FormatDiff(%+v) = %q, want %q", forcing, got, want)
	}
}
