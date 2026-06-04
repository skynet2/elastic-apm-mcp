package apm

import (
	"context"
	"fmt"
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
	Hits []eseHit `json:"hits"`
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

	var resp eseResponse
	if err := c.postRaw(ctx, "/internal/search/ese", payload, &resp); err != nil {
		return nil, fmt.Errorf("apm: ese search: %w", err)
	}

	sources := make([]map[string]any, 0, len(resp.RawResponse.Hits.Hits))
	for _, h := range resp.RawResponse.Hits.Hits {
		sources = append(sources, h.Source)
	}
	return sources, nil
}
