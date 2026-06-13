package notes

import "testing"

func TestNormalizeText(t *testing.T) {
	got := normalizeText("  hello  ")
	if got != "hello" {
		t.Fatalf("normalizeText() = %q, want %q", got, "hello")
	}
}

func TestNormalizeColor(t *testing.T) {
	if got := normalizeColor("blue"); got != "blue" {
		t.Fatalf("normalizeColor(valid) = %q, want blue", got)
	}
	if got := normalizeColor("orange"); got != "yellow" {
		t.Fatalf("normalizeColor(invalid) = %q, want yellow", got)
	}
}
