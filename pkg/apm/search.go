package apm

import (
	"context"
	"encoding/json"
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
	Hits         eseHits        `json:"hits"`
	Aggregations map[string]any `json:"aggregations"`
}

type eseHits struct {
	Total eseTotal `json:"total"`
	Hits  []eseHit `json:"hits"`
}

type eseTotal struct {
	Value int
}

// UnmarshalJSON tolerates both shapes Elasticsearch/Kibana use for hits.total:
// a bare number (e.g. `42`) and the tracked object `{"value":42,"relation":"eq"}`.
func (t *eseTotal) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		t.Value = n
		return nil
	}
	var obj struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	t.Value = obj.Value
	return nil
}

type eseHit struct {
	Index  string         `json:"_index"`
	Source map[string]any `json:"_source"`
}

// SearchResult is the full shape of an Elasticsearch search, exposing the
// matched hit count, per-hit documents (with `_index` merged into each source),
// and any aggregations. Typed callers use eseSearch for just the hits; the raw
// es_search tool returns this whole structure so aggregations and index
// provenance survive.
type SearchResult struct {
	Total        int              `json:"total"`
	Hits         []map[string]any `json:"hits"`
	Aggregations map[string]any   `json:"aggregations,omitempty"`
}

func (c *Client) eseSearchResult(ctx context.Context, index string, body map[string]any) (SearchResult, error) {
	payload := eseRequest{
		Params: eseParams{
			Index: index,
			Body:  body,
		},
	}

	var resp eseResponse
	if err := c.postRaw(ctx, "/internal/search/ese", payload, &resp); err != nil {
		return SearchResult{}, fmt.Errorf("apm: ese search: %w", err)
	}

	hits := make([]map[string]any, 0, len(resp.RawResponse.Hits.Hits))
	for _, h := range resp.RawResponse.Hits.Hits {
		hits = append(hits, mergeIndex(h.Source, h.Index))
	}

	return SearchResult{
		Total:        resp.RawResponse.Hits.Total.Value,
		Hits:         hits,
		Aggregations: resp.RawResponse.Aggregations,
	}, nil
}

func (c *Client) eseSearch(ctx context.Context, index string, body map[string]any) ([]map[string]any, error) {
	res, err := c.eseSearchResult(ctx, index, body)
	if err != nil {
		return nil, err
	}
	return res.Hits, nil
}

// mergeIndex stamps the concrete `_index` a hit came from onto its source so an
// agent can tell which index (e.g. logs-apm* vs fluent-bit-*) produced a
// document. An empty index or nil source is left untouched.
func mergeIndex(source map[string]any, index string) map[string]any {
	if index == "" || source == nil {
		return source
	}
	source["_index"] = index
	return source
}
