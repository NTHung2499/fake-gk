package router

import (
	"context"
	"errors"
	"testing"

	"github.com/nthung2499/fake-gk/internal/openai"
)

type fakeClient struct {
	output string
	err    error
	model  string
}

func (f *fakeClient) Generate(ctx context.Context, apiKey, model, instructions string, messages []openai.Message) (string, error) {
	f.model = model
	return f.output, f.err
}

func TestParseDecision(t *testing.T) {
	decision, err := ParseDecision(`extra {"route":"deep","reason":"needs architecture"} text`)
	if err != nil {
		t.Fatalf("ParseDecision() error = %v", err)
	}
	if decision.Route != RouteDeep || decision.Reason != "needs architecture" {
		t.Fatalf("ParseDecision() = %+v", decision)
	}
}

func TestParseDecisionRejectsInvalidOutput(t *testing.T) {
	if _, err := ParseDecision(`{"route":"maybe"}`); err == nil {
		t.Fatal("ParseDecision() accepted invalid route")
	}
}

func TestDecideRoutesDeep(t *testing.T) {
	client := &fakeClient{output: `{"route":"deep","reason":"debugging architecture"}`}
	r := New(client, Config{FastModel: "fast-model", DeepModel: "deep-model", RouterModel: "router-model"})

	decision := r.Decide(context.Background(), "sk-test", "Design ArgoCD GitOps architecture with HPA and DB tradeoffs", nil)
	if decision.Route != RouteDeep || decision.Model != "deep-model" {
		t.Fatalf("Decide() = %+v", decision)
	}
	if client.model != "router-model" {
		t.Fatalf("classifier model = %q", client.model)
	}
}

func TestDecideFallsBackFastOnClassifierFailure(t *testing.T) {
	client := &fakeClient{err: errors.New("classifier down")}
	r := New(client, Config{FastModel: "fast-model", DeepModel: "deep-model", RouterModel: "router-model"})

	decision := r.Decide(context.Background(), "sk-test", "hello", nil)
	if decision.Route != RouteFast || decision.Model != "fast-model" {
		t.Fatalf("Decide() = %+v", decision)
	}
}

func TestDecideRoutesFastGreeting(t *testing.T) {
	client := &fakeClient{output: `{"route":"fast","reason":"simple greeting"}`}
	r := New(client, Config{FastModel: "fast-model", DeepModel: "deep-model", RouterModel: "router-model"})

	decision := r.Decide(context.Background(), "sk-test", "chao", nil)
	if decision.Route != RouteFast || decision.Model != "fast-model" {
		t.Fatalf("Decide() = %+v", decision)
	}
}
