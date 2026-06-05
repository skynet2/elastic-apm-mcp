package apm

import (
	"context"
	"fmt"
	"net/url"
)

type TraceParams struct {
	Start              string
	End                string
	EntryTransactionID string
}

func (c *Client) TraceGet(ctx context.Context, traceID string, p TraceParams) (Trace, error) {
	q := url.Values{}
	q.Set("start", p.Start)
	q.Set("end", p.End)
	if p.EntryTransactionID != "" {
		q.Set("entryTransactionId", p.EntryTransactionID)
	}
	q.Set("ecsOnly", "true")

	path := "/internal/apm/unified_traces/" + url.PathEscape(traceID)
	var result Trace
	if err := c.get(ctx, path, q, &result); err != nil {
		return Trace{}, fmt.Errorf("trace get: %w", err)
	}
	return result, nil
}
