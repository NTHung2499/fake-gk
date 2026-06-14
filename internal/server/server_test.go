package server

import "testing"

func TestLoadTemplates(t *testing.T) {
	if _, err := loadTemplates(); err != nil {
		t.Fatalf("loadTemplates() error = %v", err)
	}
}

func TestTitleFromMessageKeepsUnicode(t *testing.T) {
	got := titleFromMessage("chào bạn")
	if got != "chào bạn" {
		t.Fatalf("titleFromMessage() = %q", got)
	}
}
