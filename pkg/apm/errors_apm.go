package apm

import (
	"context"
	"fmt"
	"net/url"
)

type ErrorGroupsParams struct {
	Service     string
	Environment string
	Kuery       string
	Start       string
	End         string
}

func (c *Client) ErrorGroups(ctx context.Context, p ErrorGroupsParams) (map[string]any, error) {
	q := url.Values{}
	q.Set("environment", orDefault(p.Environment, "ENVIRONMENT_ALL"))
	q.Set("start", p.Start)
	q.Set("end", p.End)
	q.Set("kuery", p.Kuery)

	path := "/internal/apm/services/" + url.PathEscape(p.Service) + "/errors/groups/main_statistics"
	var result map[string]any
	if err := c.get(ctx, path, q, &result); err != nil {
		return nil, fmt.Errorf("error groups: %w", err)
	}
	return result, nil
}
