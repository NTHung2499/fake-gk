package chat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewPublicID(t *testing.T) {
	id, err := NewPublicID()
	if err != nil {
		t.Fatalf("NewPublicID() error = %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("NewPublicID() length = %d, want 32", len(id))
	}
	other, err := NewPublicID()
	if err != nil {
		t.Fatalf("NewPublicID() second error = %v", err)
	}
	if id == other {
		t.Fatal("NewPublicID() returned duplicate values")
	}
}

func TestNormalizeTitle(t *testing.T) {
	if got := normalizeTitle("  "); got != "New chat" {
		t.Fatalf("normalizeTitle(empty) = %q", got)
	}
	if got := normalizeTitle("  hello  "); got != "hello" {
		t.Fatalf("normalizeTitle(trim) = %q", got)
	}
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	if got := normalizeTitle(string(long)); len(got) != 255 {
		t.Fatalf("normalizeTitle(long) length = %d, want 255", len(got))
	}
}

func TestNormalizeRoleAndStatus(t *testing.T) {
	if got := normalizeRole("assistant"); got != RoleAssistant {
		t.Fatalf("normalizeRole(assistant) = %q", got)
	}
	if got := normalizeRole("system"); got != RoleUser {
		t.Fatalf("normalizeRole(system) = %q", got)
	}
	if got := normalizeStatus("error"); got != StatusError {
		t.Fatalf("normalizeStatus(error) = %q", got)
	}
	if got := normalizeStatus("weird"); got != StatusComplete {
		t.Fatalf("normalizeStatus(weird) = %q", got)
	}
}

func TestEmptyMessagesMarshalAsArray(t *testing.T) {
	messages := make([]Message, 0)
	payload, err := json.Marshal(struct {
		Messages []Message `json:"messages"`
	}{Messages: messages})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "null") {
		t.Fatalf("empty messages marshaled as null: %s", payload)
	}
	if !strings.Contains(string(payload), `"messages":[]`) {
		t.Fatalf("empty messages did not marshal as array: %s", payload)
	}
}
