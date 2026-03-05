package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey          string
	model           string
	baseURL         string
	httpClient      *http.Client
	maxOutputTokens int
}

func NewOpenAIClient(apiKey, model, baseURL string, httpClient *http.Client, maxOutputTokens int) *OpenAIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &OpenAIClient{
		apiKey:          apiKey,
		model:           model,
		baseURL:         baseURL,
		httpClient:      httpClient,
		maxOutputTokens: maxOutputTokens,
	}
}

func (c *OpenAIClient) GenerateAnalysis(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", fmt.Errorf("openai api key is empty")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("prompt is empty")
	}

	reqBody := openAIRequest{
		Model:           c.model,
		Input:           prompt,
		MaxOutputTokens: c.maxOutputTokens,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai http %d: %s", resp.StatusCode, string(body))
	}

	var decoded openAIResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return "", err
	}
	if decoded.Error != nil && decoded.Error.Message != "" {
		return "", fmt.Errorf("openai error: %s", decoded.Error.Message)
	}
	if strings.TrimSpace(decoded.OutputText) != "" {
		return strings.TrimSpace(decoded.OutputText), nil
	}

	for _, out := range decoded.Output {
		for _, c := range out.Content {
			if strings.TrimSpace(c.Text) != "" {
				return strings.TrimSpace(c.Text), nil
			}
		}
	}
	return "", fmt.Errorf("openai response missing output_text")
}

type openAIRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
}

type openAIResponse struct {
	OutputText string           `json:"output_text"`
	Output     []openAIOutput   `json:"output"`
	Error      *openAIErrorBody `json:"error"`
}

type openAIOutput struct {
	Content []openAIContent `json:"content"`
}

type openAIContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIErrorBody struct {
	Message string `json:"message"`
}
