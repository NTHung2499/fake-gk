package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.openai.com/v1"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func NewClientWithHTTP(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

type responseRequest struct {
	Model        string    `json:"model"`
	Instructions string    `json:"instructions,omitempty"`
	Input        []Message `json:"input"`
	Stream       bool      `json:"stream,omitempty"`
}

func (c *Client) Generate(ctx context.Context, apiKey, model, instructions string, messages []Message) (string, error) {
	payload := responseRequest{
		Model:        model,
		Instructions: instructions,
		Input:        messages,
	}

	body, err := c.do(ctx, apiKey, payload)
	if err != nil {
		return "", err
	}
	defer body.Close()

	var response struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.OutputText) != "" {
		return response.OutputText, nil
	}
	var builder strings.Builder
	for _, item := range response.Output {
		for _, content := range item.Content {
			if content.Text != "" {
				builder.WriteString(content.Text)
			}
		}
	}
	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", errors.New("OpenAI returned an empty response")
	}
	return text, nil
}

func (c *Client) Stream(ctx context.Context, apiKey, model, instructions string, messages []Message, onDelta func(string) error) (string, error) {
	payload := responseRequest{
		Model:        model,
		Instructions: instructions,
		Input:        messages,
		Stream:       true,
	}

	body, err := c.do(ctx, apiKey, payload)
	if err != nil {
		return "", err
	}
	defer body.Close()

	var builder strings.Builder
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		delta, err := parseDelta(data)
		if err != nil {
			return builder.String(), err
		}
		if delta == "" {
			continue
		}
		builder.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return builder.String(), err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return builder.String(), err
	}
	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", errors.New("OpenAI returned an empty response")
	}
	return text, nil
}

func (c *Client) do(ctx context.Context, apiKey string, payload responseRequest) (io.ReadCloser, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("OpenAI API key is required")
	}
	if strings.TrimSpace(payload.Model) == "" {
		return nil, errors.New("OpenAI model is required")
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	req.Header.Set("Content-Type", "application/json")
	if payload.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.Body, nil
	}
	defer resp.Body.Close()
	return nil, mapError(resp.StatusCode, resp.Body)
}

func parseDelta(data string) (string, error) {
	var event struct {
		Type  string `json:"type"`
		Delta string `json:"delta"`
		Text  string `json:"text"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return "", err
	}
	if event.Error != nil && event.Error.Message != "" {
		return "", errors.New(event.Error.Message)
	}
	switch event.Type {
	case "response.output_text.delta":
		return event.Delta, nil
	case "response.refusal.delta":
		return event.Delta, nil
	case "response.completed", "response.created", "response.in_progress", "response.output_item.added", "response.content_part.added", "response.content_part.done", "response.output_item.done":
		return "", nil
	default:
		if event.Delta != "" {
			return event.Delta, nil
		}
		if event.Text != "" {
			return event.Text, nil
		}
		return "", nil
	}
}

func mapError(status int, body io.Reader) error {
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	_ = json.NewDecoder(body).Decode(&payload)
	message := strings.TrimSpace(payload.Error.Message)
	if message == "" {
		message = http.StatusText(status)
	}

	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("OpenAI rejected this API key: %s", message)
	case http.StatusTooManyRequests:
		return fmt.Errorf("OpenAI rate limit reached: %s", message)
	default:
		if status >= 500 {
			return fmt.Errorf("OpenAI service error: %s", message)
		}
		return fmt.Errorf("OpenAI request failed: %s", message)
	}
}
