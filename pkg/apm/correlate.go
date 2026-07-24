package apm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var simpleKueryFieldRe = regexp.MustCompile(`^[A-Za-z_@][A-Za-z0-9_.@-]*$`)

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
	if strings.TrimSpace(kuery) == "" {
		return filters
	}
	if field, value, ok := parseSimpleKuery(kuery); ok {
		return append(filters, map[string]any{
			"match": map[string]any{field: value},
		})
	}
	return append(filters, map[string]any{
		"query_string": map[string]any{
			"query":            kuery,
			"analyze_wildcard": true,
		},
	})
}

func parseSimpleKuery(kuery string) (string, string, bool) {
	k := strings.TrimSpace(kuery)
	idx := strings.Index(k, ":")
	if idx <= 0 {
		return "", "", false
	}
	field := strings.TrimSpace(k[:idx])
	value := strings.TrimSpace(k[idx+1:])
	if !simpleKueryFieldRe.MatchString(field) {
		return "", "", false
	}
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		inner := value[1 : len(value)-1]
		if inner == "" || strings.Contains(inner, `"`) {
			return "", "", false
		}
		return field, inner, true
	}
	if value == "" || strings.ContainsAny(value, " :\"*()<>") {
		return "", "", false
	}
	return field, value, true
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

// addMatchPhraseFilter matches a text-analyzed field (e.g. fluent-bit's
// service_name / k8_pod_name) by phrase, so a caller-supplied value like
// "bloody-items" matches the tokenized field without needing query_string
// wildcards. A term query would silently miss on these fields.
func addMatchPhraseFilter(filters []map[string]any, field, value string) []map[string]any {
	if value == "" {
		return filters
	}
	return append(filters, map[string]any{
		"match_phrase": map[string]any{field: value},
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

func (c *Client) RawSearch(ctx context.Context, index string, body map[string]any) (SearchResult, error) {
	return c.eseSearchResult(ctx, index, body)
}

type AppLogsParams struct {
	Service       string
	Pod           string
	Namespace     string
	Container     string
	TraceID       string
	TransactionID string
	Message       string
	Level         string
	Kuery         string
	Start         string
	End           string
	Size          int
}

// AppLogsSearch queries the application/container stdout log stream shipped by
// fluent-bit. That stream uses a flattened, non-ECS schema (service_name,
// k8_pod_name, level, message, trace_id, transaction_id, plus arbitrary
// structured fields promoted to the top level) and is a different index family
// than the APM error logs LogsSearch targets. Text-analyzed fields
// (service_name, k8_pod_name, k8_namespace, k8_container_name) are matched with
// match_phrase so callers pass a plain value rather than crafting query_string
// wildcards; the keyword correlation ids (trace_id, transaction_id) use term.
func (c *Client) AppLogsSearch(ctx context.Context, p AppLogsParams) ([]map[string]any, error) {
	var filters []map[string]any
	filters = addMatchPhraseFilter(filters, "service_name", p.Service)
	filters = addMatchPhraseFilter(filters, "k8_pod_name", p.Pod)
	filters = addMatchPhraseFilter(filters, "k8_namespace", p.Namespace)
	filters = addMatchPhraseFilter(filters, "k8_container_name", p.Container)
	filters = addTermFilter(filters, "trace_id", p.TraceID)
	filters = addTermFilter(filters, "transaction_id", p.TransactionID)
	filters = addMatchPhraseFilter(filters, "message", p.Message)
	filters = addMatchPhraseFilter(filters, "level", p.Level)
	filters = addKueryFilter(filters, p.Kuery)
	filters = addTimeRangeFilter(filters, p.Start, p.End)

	sort := []map[string]any{{"@timestamp": "desc"}}
	size := p.Size
	if size <= 0 {
		size = 50
	}
	body := buildBoolQuery(filters, sort, size)
	return c.eseSearch(ctx, c.appLogsIndex, body)
}

type TraceLogsParams struct {
	TraceID string
	Start   string
	End     string
	Size    int
}

// TraceLogs assembles the full log picture for a trace in one call: APM error
// documents (keyed by ECS trace.id) and fluent-bit application logs (keyed by
// the flattened trace_id) merged into a single timeline sorted newest-first.
// Each entry keeps its `_index` so the source stream stays identifiable.
func (c *Client) TraceLogs(ctx context.Context, p TraceLogsParams) ([]map[string]any, error) {
	if p.TraceID == "" {
		return nil, fmt.Errorf("trace_logs: traceId required")
	}

	size := p.Size
	if size <= 0 {
		size = 100
	}

	apmFilters := addTimeRangeFilter(
		addTermFilter(nil, "trace.id", p.TraceID), p.Start, p.End)
	appFilters := addTimeRangeFilter(
		addTermFilter(nil, "trace_id", p.TraceID), p.Start, p.End)

	sort := []map[string]any{{"@timestamp": "desc"}}

	apmLogs, err := c.eseSearch(ctx, "logs-apm*", buildBoolQuery(apmFilters, sort, size))
	if err != nil {
		return nil, err
	}
	appLogs, err := c.eseSearch(ctx, c.appLogsIndex, buildBoolQuery(appFilters, sort, size))
	if err != nil {
		return nil, err
	}

	merged := make([]map[string]any, 0, len(apmLogs)+len(appLogs))
	merged = append(merged, apmLogs...)
	merged = append(merged, appLogs...)
	sortByTimestampDesc(merged)
	if len(merged) > size {
		merged = merged[:size]
	}
	return merged, nil
}
