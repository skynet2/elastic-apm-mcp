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
	setCommonParams(q, p.Environment, p.Start, p.End, p.Kuery)

	path := "/internal/apm/services/" + url.PathEscape(p.Service) + "/errors/groups/main_statistics"
	var result map[string]any
	if err := c.get(ctx, path, q, &result); err != nil {
		return nil, fmt.Errorf("error groups: %w", err)
	}
	return result, nil
}
