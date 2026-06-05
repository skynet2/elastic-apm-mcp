package mcp

import (
	"context"

	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

//go:generate mockgen -source=client.go -destination=mocks/mock_client.go -package=mocks

// APMClient defines the subset of apm.Client used by MCP handlers.
type APMClient interface {
	ServiceList(ctx context.Context, p apm.ServiceListParams) ([]apm.Service, error)
	TransactionGroups(ctx context.Context, p apm.TransactionGroupsParams) ([]apm.TransactionGroup, error)
	TransactionSamples(ctx context.Context, p apm.TransactionSamplesParams) ([]apm.TraceSample, error)
	ServiceMetrics(ctx context.Context, p apm.ServiceMetricsParams) (map[string]any, error)
	TraceGet(ctx context.Context, traceID string, p apm.TraceParams) (apm.Trace, error)
	ErrorGroups(ctx context.Context, p apm.ErrorGroupsParams) (map[string]any, error)
	ErrorGet(ctx context.Context, p apm.ErrorGetParams) ([]map[string]any, error)
	LogsSearch(ctx context.Context, p apm.LogsParams) ([]map[string]any, error)
	TraceSearch(ctx context.Context, p apm.TraceSearchParams) ([]map[string]any, error)
	ServiceDependencies(ctx context.Context, p apm.DependenciesParams) ([]apm.ServiceDependency, error)
	Environments(ctx context.Context, start, end string) ([]string, error)
	GetAPMIndices(ctx context.Context) (apm.APMIndices, error)
	RawSearch(ctx context.Context, index string, body map[string]any) ([]map[string]any, error)
}

var _ APMClient = (*apm.Client)(nil)
