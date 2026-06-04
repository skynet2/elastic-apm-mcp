package apm

import (
	"context"
	"fmt"
	"net/url"
)

type TransactionGroupsParams struct {
	Service         string
	Environment     string
	Kuery           string
	Start           string
	End             string
	TransactionType string
}

type transactionGroupsResponse struct {
	TransactionGroups []TransactionGroup `json:"transactionGroups"`
}

func (c *Client) TransactionGroups(ctx context.Context, p TransactionGroupsParams) ([]TransactionGroup, error) {
	q := url.Values{}
	q.Set("environment", orDefault(p.Environment, "ENVIRONMENT_ALL"))
	q.Set("start", p.Start)
	q.Set("end", p.End)
	q.Set("kuery", p.Kuery)
	q.Set("transactionType", orDefault(p.TransactionType, "request"))
	q.Set("latencyAggregationType", "avg")
	q.Set("documentType", "transactionMetric")
	q.Set("rollupInterval", "1m")
	q.Set("useDurationSummary", "true")

	path := "/internal/apm/services/" + url.PathEscape(p.Service) + "/transactions/groups/main_statistics"
	var resp transactionGroupsResponse
	if err := c.get(ctx, path, q, &resp); err != nil {
		return nil, fmt.Errorf("transaction groups: %w", err)
	}
	return resp.TransactionGroups, nil
}

type TransactionSamplesParams struct {
	Service         string
	Environment     string
	Kuery           string
	Start           string
	End             string
	TransactionType string
	TransactionName string
}

type transactionSamplesResponse struct {
	TraceSamples []TraceSample `json:"traceSamples"`
}

func (c *Client) TransactionSamples(ctx context.Context, p TransactionSamplesParams) ([]TraceSample, error) {
	q := url.Values{}
	q.Set("environment", orDefault(p.Environment, "ENVIRONMENT_ALL"))
	q.Set("start", p.Start)
	q.Set("end", p.End)
	q.Set("kuery", p.Kuery)
	q.Set("transactionType", orDefault(p.TransactionType, "request"))
	q.Set("transactionName", p.TransactionName)

	path := "/internal/apm/services/" + url.PathEscape(p.Service) + "/transactions/traces/samples"
	var resp transactionSamplesResponse
	if err := c.get(ctx, path, q, &resp); err != nil {
		return nil, fmt.Errorf("transaction samples: %w", err)
	}
	return resp.TraceSamples, nil
}
