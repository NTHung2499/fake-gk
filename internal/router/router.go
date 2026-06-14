package router

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/nthung2499/fake-gk/internal/openai"
)

const (
	RouteFast = "fast"
	RouteDeep = "deep"
)

const classifierInstructions = `Classify the user's latest message for model routing.
Return only compact JSON with this shape: {"route":"fast|deep","reason":"short reason"}.
Use "fast" for greetings, simple Q&A, FAQs, casual clarification, short translation, and simple non-technical prompts.
Use "deep" for multi-step reasoning, debugging, architecture/design, Kubernetes/GitOps/database planning, code review, tool-use-like tasks, long context, and ambiguous complex requests.`

type OpenAIClient interface {
	Generate(ctx context.Context, apiKey, model, instructions string, messages []openai.Message) (string, error)
}

type Config struct {
	FastModel   string
	DeepModel   string
	RouterModel string
}

type Decision struct {
	Route  string `json:"route"`
	Model  string `json:"model"`
	Reason string `json:"reason"`
}

type Router struct {
	client OpenAIClient
	cfg    Config
}

func New(client OpenAIClient, cfg Config) *Router {
	cfg.FastModel = defaultString(cfg.FastModel, "gpt-5.4-mini")
	cfg.DeepModel = defaultString(cfg.DeepModel, "gpt-5.5")
	cfg.RouterModel = defaultString(cfg.RouterModel, cfg.FastModel)
	return &Router{client: client, cfg: cfg}
}

func (r *Router) Decide(ctx context.Context, apiKey string, latestUserMessage string, recent []openai.Message) Decision {
	fallback := Decision{Route: RouteFast, Model: r.cfg.FastModel, Reason: "classifier fallback"}
	latestUserMessage = strings.TrimSpace(latestUserMessage)
	if latestUserMessage == "" {
		return fallback
	}

	input := buildClassifierInput(latestUserMessage, recent)
	output, err := r.client.Generate(ctx, apiKey, r.cfg.RouterModel, classifierInstructions, input)
	if err != nil {
		return fallback
	}

	decision, err := ParseDecision(output)
	if err != nil {
		return fallback
	}
	if decision.Route == RouteDeep {
		decision.Model = r.cfg.DeepModel
		return decision
	}
	decision.Route = RouteFast
	decision.Model = r.cfg.FastModel
	return decision
}

func ParseDecision(output string) (Decision, error) {
	output = strings.TrimSpace(output)
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start >= 0 && end >= start {
		output = output[start : end+1]
	}

	var decision Decision
	if err := json.Unmarshal([]byte(output), &decision); err != nil {
		return Decision{}, err
	}
	decision.Route = strings.ToLower(strings.TrimSpace(decision.Route))
	decision.Reason = strings.TrimSpace(decision.Reason)
	if decision.Route != RouteFast && decision.Route != RouteDeep {
		return Decision{}, errors.New("invalid route")
	}
	if decision.Reason == "" {
		decision.Reason = "classifier selected " + decision.Route
	}
	return decision, nil
}

func buildClassifierInput(latestUserMessage string, recent []openai.Message) []openai.Message {
	const maxRecent = 6
	if len(recent) > maxRecent {
		recent = recent[len(recent)-maxRecent:]
	}

	input := make([]openai.Message, 0, len(recent)+1)
	for _, message := range recent {
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		if message.Role == "user" && strings.TrimSpace(message.Content) == latestUserMessage {
			continue
		}
		input = append(input, message)
	}
	input = append(input, openai.Message{
		Role:    "user",
		Content: "Latest user message to classify:\n" + latestUserMessage,
	})
	return input
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
