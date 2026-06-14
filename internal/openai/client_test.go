package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateBuildsResponsesRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %s, want /responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload responseRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if payload.Model != "gpt-test" {
			t.Fatalf("model = %q", payload.Model)
		}
		if payload.Instructions != "be helpful" {
			t.Fatalf("instructions = %q", payload.Instructions)
		}
		if len(payload.Input) != 1 || payload.Input[0].Content != "hello" {
			t.Fatalf("input = %+v", payload.Input)
		}
		_, _ = w.Write([]byte(`{"output_text":"hi there"}`))
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, server.Client())
	got, err := client.Generate(context.Background(), "sk-test", "gpt-test", "be helpful", []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got != "hi there" {
		t.Fatalf("Generate() = %q", got)
	}
}

func TestStreamReadsDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); accept != "text/event-stream" {
			t.Fatalf("Accept = %q", accept)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hel\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"lo\"}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, server.Client())
	var deltas []string
	got, err := client.Stream(context.Background(), "sk-test", "gpt-test", "", []Message{{Role: "user", Content: "hello"}}, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if got != "Hello" {
		t.Fatalf("Stream() = %q", got)
	}
	if strings.Join(deltas, "") != "Hello" {
		t.Fatalf("deltas = %+v", deltas)
	}
}

func TestGenerateMapsOpenAIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, server.Client())
	_, err := client.Generate(context.Background(), "sk-bad", "gpt-test", "", []Message{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatal("Generate() error = nil")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("Generate() error = %q", err.Error())
	}
}
