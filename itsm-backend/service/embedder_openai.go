package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

// OpenAIEmbedder implements real embeddings call using OpenAI-compatible API
// Reads API key and endpoint from environment or injected config
// Default model: text-embedding-3-small (1536 dims)
type OpenAIEmbedder struct {
	apiKey   string
	endpoint string
	model    string
}

func NewOpenAIEmbedderWithConfig(apiKey, endpoint, model string) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/embeddings"
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAIEmbedder{apiKey: apiKey, endpoint: endpoint, model: model}
}

func NewOpenAIEmbedder() *OpenAIEmbedder { return NewOpenAIEmbedderWithConfig("", "", "") }

type embeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (e *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	if e.apiKey == "" {
		return make([]float32, 1536), nil
	}
	payload := embeddingRequest{Input: text, Model: e.model}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, e.endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, errors.New("embedding request failed")
	}
	var er embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, err
	}
	if len(er.Data) == 0 {
		return nil, errors.New("no embedding returned")
	}
	// convert []float64 -> []float32
	src := er.Data[0].Embedding
	out := make([]float32, len(src))
	for i, v := range src {
		out[i] = float32(v)
	}
	return out, nil
}
