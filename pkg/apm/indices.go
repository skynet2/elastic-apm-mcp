package apm

import (
	"context"
	"fmt"
	"net/url"
	"sort"
)

// ListIndices returns the concrete index names matching a pattern, discovered
// via a terms aggregation over the `_index` metadata field through the same
// Kibana ES proxy the search tools use. It exists so an agent can find real
// index names (e.g. that application logs live under fluent-bit-*) instead of
// guessing patterns. An empty pattern defaults to "*".
func (c *Client) ListIndices(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}

	body := map[string]any{
		"size": 0,
		"aggs": map[string]any{
			"indices": map[string]any{
				"terms": map[string]any{
					"field": "_index",
					"size":  1000,
				},
			},
		},
	}

	res, err := c.eseSearchResult(ctx, pattern, body)
	if err != nil {
		return nil, err
	}

	names := extractBucketKeys(res.Aggregations, "indices")
	sort.Strings(names)
	return names, nil
}

func extractBucketKeys(aggs map[string]any, name string) []string {
	agg, ok := aggs[name].(map[string]any)
	if !ok {
		return nil
	}
	buckets, ok := agg["buckets"].([]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(buckets))
	for _, b := range buckets {
		bucket, ok := b.(map[string]any)
		if !ok {
			continue
		}
		if key, ok := bucket["key"].(string); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

type Field struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	ESTypes      []string `json:"esTypes"`
	Searchable   bool     `json:"searchable"`
	Aggregatable bool     `json:"aggregatable"`
}

type dataViewFieldsResponse struct {
	Fields []Field `json:"fields"`
}

// DescribeFields lists the fields of an index pattern with their Elasticsearch
// types via Kibana's data-views API. Knowing a field's esType (keyword vs text)
// tells an agent whether to filter it with term or match_phrase — the
// distinction that makes term queries silently miss on analyzed fields like
// service_name / k8_pod_name.
func (c *Client) DescribeFields(ctx context.Context, pattern string) ([]Field, error) {
	if pattern == "" {
		return nil, fmt.Errorf("describe_fields: pattern required")
	}

	q := url.Values{}
	q.Set("pattern", pattern)
	q.Set("allow_no_index", "true")
	q.Set("apiVersion", "1")
	for _, meta := range []string{"_source", "_id", "_index", "_score"} {
		q.Add("meta_fields", meta)
	}

	var resp dataViewFieldsResponse
	if err := c.getRaw(ctx, "/internal/data_views/fields", q, &resp); err != nil {
		return nil, fmt.Errorf("describe_fields: %w", err)
	}
	return resp.Fields, nil
}

// sortByTimestampDesc orders documents newest-first by their `@timestamp`
// string. RFC3339 timestamps sort correctly lexicographically; entries without
// a string timestamp sort last.
func sortByTimestampDesc(docs []map[string]any) {
	sort.SliceStable(docs, func(i, j int) bool {
		return docTimestamp(docs[i]) > docTimestamp(docs[j])
	})
}

func docTimestamp(doc map[string]any) string {
	if ts, ok := doc["@timestamp"].(string); ok {
		return ts
	}
	return ""
}
