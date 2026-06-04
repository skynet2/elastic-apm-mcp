package apm

import (
	"context"
	"fmt"
)

type ErrorGetParams struct {
	ErrorID     string
	GroupingKey string
}

type LogsParams struct {
	TraceID       string
	TransactionID string
	SpanID        string
	Kuery         string
	Start         string
	End           string
	Size          int
}

type TraceSearchParams struct {
	Kuery   string
	Service string
	Start   string
	End     string
	Size    int
}

func buildBoolQuery(filters []map[string]any, sort []map[string]any, size int) map[string]any {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
	}
	if len(sort) > 0 {
		body["sort"] = sort
	}
	if size > 0 {
		body["size"] = size
	}
	return body
}

func addKueryFilter(filters []map[string]any, kuery string) []map[string]any {
	if kuery == "" {
		return filters
	}
	return append(filters, map[string]any{
		"query_string": map[string]any{
			"query":           kuery,
			"analyze_wildcard": true,
		},
	})
}

func addTimeRangeFilter(filters []map[string]any, start, end string) []map[string]any {
	if start == "" && end == "" {
		return filters
	}
	r := map[string]any{}
	if start != "" {
		r["gte"] = start
	}
	if end != "" {
		r["lte"] = end
	}
	return append(filters, map[string]any{
		"range": map[string]any{"@timestamp": r},
	})
}

func addTermFilter(filters []map[string]any, field, value string) []map[string]any {
	if value == "" {
		return filters
	}
	return append(filters, map[string]any{
		"term": map[string]any{field: value},
	})
}

func (c *Client) ErrorGet(ctx context.Context, p ErrorGetParams) ([]map[string]any, error) {
	if p.ErrorID == "" && p.GroupingKey == "" {
		return nil, fmt.Errorf("error_get: errorId or groupingKey required")
	}

	var filters []map[string]any
	sort := []map[string]any{{"@timestamp": "desc"}}
	size := 1

	if p.ErrorID != "" {
		filters = addTermFilter(filters, "error.id", p.ErrorID)
		sort = nil
		size = 0
	} else {
		filters = addTermFilter(filters, "error.grouping_key", p.GroupingKey)
	}

	body := buildBoolQuery(filters, sort, size)
	return c.eseSearch(ctx, "logs-apm*", body)
}

func (c *Client) LogsSearch(ctx context.Context, p LogsParams) ([]map[string]any, error) {
	var filters []map[string]any
	filters = addTermFilter(filters, "trace.id", p.TraceID)
	filters = addTermFilter(filters, "transaction.id", p.TransactionID)
	filters = addTermFilter(filters, "span.id", p.SpanID)
	filters = addKueryFilter(filters, p.Kuery)
	filters = addTimeRangeFilter(filters, p.Start, p.End)

	sort := []map[string]any{{"@timestamp": "desc"}}
	size := p.Size
	if size <= 0 {
		size = 50
	}
	body := buildBoolQuery(filters, sort, size)
	return c.eseSearch(ctx, "logs-apm*,logs-*", body)
}

func (c *Client) TraceSearch(ctx context.Context, p TraceSearchParams) ([]map[string]any, error) {
	filters := []map[string]any{
		{"term": map[string]any{"processor.event": "transaction"}},
	}
	filters = addTermFilter(filters, "service.name", p.Service)
	filters = addKueryFilter(filters, p.Kuery)
	filters = addTimeRangeFilter(filters, p.Start, p.End)

	sort := []map[string]any{{"@timestamp": "desc"}}
	size := p.Size
	if size <= 0 {
		size = 50
	}
	body := buildBoolQuery(filters, sort, size)
	return c.eseSearch(ctx, "traces-apm*", body)
}

func (c *Client) RawSearch(ctx context.Context, index string, body map[string]any) ([]map[string]any, error) {
	return c.eseSearch(ctx, index, body)
}
