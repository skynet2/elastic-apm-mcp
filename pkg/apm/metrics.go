package apm

import (
	"context"
	"fmt"
	"net/url"
)

type ServiceMetricsParams struct {
	Service         string
	Metric          string
	Environment     string
	Kuery           string
	Start           string
	End             string
	TransactionType string
	TransactionName string
	Offset          string
}

func (c *Client) ServiceMetrics(ctx context.Context, p ServiceMetricsParams) (map[string]any, error) {
	svc := url.PathEscape(p.Service)
	env := orDefault(p.Environment, "ENVIRONMENT_ALL")
	txType := orDefault(p.TransactionType, "request")

	var path string
	q := url.Values{}

	switch p.Metric {
	case "latency":
		path = "/internal/apm/services/" + svc + "/transactions/charts/latency"
		q.Set("environment", env)
		q.Set("start", p.Start)
		q.Set("end", p.End)
		q.Set("kuery", p.Kuery)
		q.Set("transactionType", txType)
		q.Set("useDurationSummary", "true")
		if p.TransactionName != "" {
			q.Set("transactionName", p.TransactionName)
		}
		q.Set("latencyAggregationType", "avg")
		if p.Offset != "" {
			q.Set("offset", p.Offset)
		}
		q.Set("documentType", "transactionMetric")
		q.Set("rollupInterval", "1m")
		q.Set("bucketSizeInSeconds", "60")

	case "throughput":
		path = "/internal/apm/services/" + svc + "/throughput"
		q.Set("environment", env)
		q.Set("start", p.Start)
		q.Set("end", p.End)
		q.Set("kuery", p.Kuery)
		q.Set("transactionType", txType)
		if p.Offset != "" {
			q.Set("offset", p.Offset)
		}
		if p.TransactionName != "" {
			q.Set("transactionName", p.TransactionName)
		}
		q.Set("documentType", "transactionMetric")
		q.Set("rollupInterval", "1m")
		q.Set("bucketSizeInSeconds", "60")

	case "error_rate":
		path = "/internal/apm/services/" + svc + "/transactions/charts/error_rate"
		q.Set("environment", env)
		q.Set("start", p.Start)
		q.Set("end", p.End)
		q.Set("kuery", p.Kuery)
		q.Set("transactionType", txType)
		if p.TransactionName != "" {
			q.Set("transactionName", p.TransactionName)
		}
		if p.Offset != "" {
			q.Set("offset", p.Offset)
		}
		q.Set("documentType", "transactionMetric")
		q.Set("rollupInterval", "1m")
		q.Set("bucketSizeInSeconds", "60")

	case "breakdown":
		path = "/internal/apm/services/" + svc + "/transaction/charts/breakdown"
		q.Set("environment", env)
		q.Set("start", p.Start)
		q.Set("end", p.End)
		q.Set("kuery", p.Kuery)
		if p.TransactionName != "" {
			q.Set("transactionName", p.TransactionName)
		}
		q.Set("transactionType", txType)

	default:
		return nil, fmt.Errorf("unsupported metric %q", p.Metric)
	}

	var result map[string]any
	if err := c.get(ctx, path, q, &result); err != nil {
		return nil, fmt.Errorf("service metrics: %w", err)
	}
	return result, nil
}
