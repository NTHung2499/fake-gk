package server

import "testing"

func TestLoadTemplates(t *testing.T) {
	if _, err := loadTemplates(); err != nil {
		t.Fatalf("loadTemplates() error = %v", err)
	}
}
