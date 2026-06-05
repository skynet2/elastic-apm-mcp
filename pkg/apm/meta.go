package apm

import (
	"context"
	"fmt"
	"net/url"
)

type DependenciesParams struct {
	Service     string
	Environment string
	Start       string
	End         string
	Offset      string
}

type serviceDependenciesResponse struct {
	ServiceDependencies []ServiceDependency `json:"serviceDependencies"`
}

func (c *Client) ServiceDependencies(ctx context.Context, p DependenciesParams) ([]ServiceDependency, error) {
	q := url.Values{}
	q.Set("environment", orDefault(p.Environment, "ENVIRONMENT_ALL"))
	q.Set("start", p.Start)
	q.Set("end", p.End)
	q.Set("numBuckets", "20")
	if p.Offset != "" {
		q.Set("offset", p.Offset)
	}

	path := "/internal/apm/services/" + url.PathEscape(p.Service) + "/dependencies"
	var resp serviceDependenciesResponse
	if err := c.get(ctx, path, q, &resp); err != nil {
		return nil, fmt.Errorf("service dependencies: %w", err)
	}
	return resp.ServiceDependencies, nil
}

type environmentsResponse struct {
	Environments []string `json:"environments"`
}

func (c *Client) Environments(ctx context.Context, start, end string) ([]string, error) {
	q := url.Values{}
	q.Set("start", start)
	q.Set("end", end)

	var resp environmentsResponse
	if err := c.get(ctx, "/internal/apm/environments", q, &resp); err != nil {
		return nil, fmt.Errorf("environments: %w", err)
	}
	return resp.Environments, nil
}

func (c *Client) GetAPMIndices(ctx context.Context) (APMIndices, error) {
	var result APMIndices
	if err := c.get(ctx, "/internal/apm-sources/settings/apm-indices", nil, &result); err != nil {
		return APMIndices{}, fmt.Errorf("apm indices: %w", err)
	}
	return result, nil
}
