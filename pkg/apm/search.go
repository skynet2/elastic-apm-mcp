package apm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type eseRequest struct {
	Params eseParams `json:"params"`
}

type eseParams struct {
	Index string         `json:"index"`
	Body  map[string]any `json:"body"`
}

type eseResponse struct {
	RawResponse eseRawResponse `json:"rawResponse"`
}

type eseRawResponse struct {
	Hits eseHits `json:"hits"`
}

type eseHits struct {
	Total int      `json:"total"`
	Hits  []eseHit `json:"hits"`
}

type eseHit struct {
	Source map[string]any `json:"_source"`
}

func (c *Client) eseSearch(ctx context.Context, index string, body map[string]any) ([]map[string]any, error) {
	payload := eseRequest{
		Params: eseParams{
			Index: index,
			Body:  body,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("apm: ese marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/search/ese", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("apm: ese build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp eseResponse
	if err := c.do(req, &resp); err != nil {
		return nil, fmt.Errorf("apm: ese search: %w", err)
	}

	sources := make([]map[string]any, 0, len(resp.RawResponse.Hits.Hits))
	for _, h := range resp.RawResponse.Hits.Hits {
		sources = append(sources, h.Source)
	}
	return sources, nil
}
