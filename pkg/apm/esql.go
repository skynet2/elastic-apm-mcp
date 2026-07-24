package apm

import (
	"context"
	"fmt"
	"strings"
)

type ESQLColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ESQLResult is a decoded ES|QL response. Columns preserves the native column
// order and types; Rows maps each result tuple to {column: value} for readable
// consumption.
type ESQLResult struct {
	Columns  []ESQLColumn     `json:"columns"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
}

type esqlRequest struct {
	Params esqlParams `json:"params"`
}

type esqlParams struct {
	Query string `json:"query"`
}

type esqlResponse struct {
	RawResponse esqlRawResponse `json:"rawResponse"`
}

type esqlRawResponse struct {
	Columns []ESQLColumn `json:"columns"`
	Values  [][]any      `json:"values"`
}

// ESQL runs a raw ES|QL pipeline query (FROM ... | WHERE ... | STATS ...) via
// Kibana's ES|QL search strategy. Unlike es_search (Query DSL) it supports the
// piped language directly, which is the practical way to do cross-dataset
// aggregation (COUNT/VALUES ... BY) that DSL cannot express in one call. Scope
// the time range inside the query (WHERE @timestamp > NOW() - 1 hour); heavy
// unbounded scans can exceed the request timeout.
func (c *Client) ESQL(ctx context.Context, query string) (ESQLResult, error) {
	if strings.TrimSpace(query) == "" {
		return ESQLResult{}, fmt.Errorf("esql: query required")
	}

	var resp esqlResponse
	if err := c.postRaw(ctx, "/internal/search/esql", esqlRequest{Params: esqlParams{Query: query}}, &resp); err != nil {
		return ESQLResult{}, fmt.Errorf("esql: %w", err)
	}

	raw := resp.RawResponse
	rows := make([]map[string]any, 0, len(raw.Values))
	for _, values := range raw.Values {
		row := make(map[string]any, len(raw.Columns))
		for i, col := range raw.Columns {
			if i < len(values) {
				row[col.Name] = values[i]
			}
		}
		rows = append(rows, row)
	}

	return ESQLResult{
		Columns:  raw.Columns,
		Rows:     rows,
		RowCount: len(rows),
	}, nil
}
