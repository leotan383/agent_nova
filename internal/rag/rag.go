package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/store"
)

type Indexer struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewIndexer(cfg *config.Config) *Indexer {
	base := strings.TrimRight(cfg.OpenAIBaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &Indexer{
		apiKey:  cfg.OpenAIAPIKey,
		baseURL: base,
		model:   "text-embedding-3-small",
		client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

func (idx *Indexer) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{
		"model": idx.model,
		"input": text,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, idx.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+idx.apiKey)
	resp, err := idx.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings api %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return out.Data[0].Embedding, nil
}

func (idx *Indexer) IndexText(ctx context.Context, st *store.Store, kind, refID, text string) error {
	vec, err := idx.Embed(ctx, text)
	if err != nil {
		return err
	}
	return st.UpsertEmbedding(store.Embedding{
		ID: store.EmbeddingID(kind, refID), Kind: kind, RefID: refID, Text: truncate(text, 500), Vector: vec,
	})
}

func (idx *Indexer) Search(ctx context.Context, st *store.Store, query string, limit int) ([]store.Embedding, []float64, error) {
	vec, err := idx.Embed(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	return st.SearchEmbeddings(vec, limit)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
