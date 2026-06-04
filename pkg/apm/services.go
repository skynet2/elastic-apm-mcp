package apm

import (
	"context"
	"fmt"
	"net/url"
)

type ServiceListParams struct {
	Environment string
	Kuery       string
	Start       string
	End         string
}

type serviceListResponse struct {
	Items []Service `json:"items"`
}

func (c *Client) ServiceList(ctx context.Context, p ServiceListParams) ([]Service, error) {
	q := url.Values{}
	setCommonParams(q, p.Environment, p.Start, p.End, p.Kuery)
	q.Set("documentType", "serviceTransactionMetric")
	q.Set("rollupInterval", "1m")
	q.Set("useDurationSummary", "true")
	q.Set("probability", "1")
	q.Set("serviceGroup", "")
	q.Set("searchQuery", "")

	var resp serviceListResponse
	if err := c.get(ctx, "/internal/apm/services", q, &resp); err != nil {
		return nil, fmt.Errorf("service list: %w", err)
	}
	return resp.Items, nil
}
