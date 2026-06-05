package mcp

import (
	"context"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

func registerServiceTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger, now func() time.Time) {
	s.AddTool(mcplib.NewTool("service_list",
		mcplib.WithDescription("Lists all APM services with latency, throughput, and error rate. Returns serviceName and environments for correlation."),
		mcplib.WithString("environment", mcplib.Description("Filter by environment")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time (ISO8601 or relative, e.g. now-15m)")),
		mcplib.WithString("end", mcplib.Description("End time (ISO8601 or relative, e.g. now)")),
	), serviceListHandler(c, log, now))

	s.AddTool(mcplib.NewTool("service_metrics",
		mcplib.WithDescription("Returns latency, throughput, error_rate, or breakdown time series for a service. Use service+metric to chart performance trends."),
		mcplib.WithString("service", mcplib.Required(), mcplib.Description("Service name")),
		mcplib.WithString("metric", mcplib.Required(), mcplib.Description("One of: latency, throughput, error_rate, breakdown")),
		mcplib.WithString("environment", mcplib.Description("Filter by environment")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithString("transaction_type", mcplib.Description("Transaction type (default: request)")),
		mcplib.WithString("transaction_name", mcplib.Description("Transaction name filter")),
		mcplib.WithString("offset", mcplib.Description("Comparison offset (e.g. 1d)")),
	), serviceMetricsHandler(c, log, now))

	s.AddTool(mcplib.NewTool("service_dependencies",
		mcplib.WithDescription("Lists downstream dependencies of a service with stats. Returns dependencyName and span type for pivot."),
		mcplib.WithString("service", mcplib.Required(), mcplib.Description("Service name")),
		mcplib.WithString("environment", mcplib.Description("Filter by environment")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithString("offset", mcplib.Description("Comparison offset")),
	), serviceDependenciesHandler(c, log, now))

	s.AddTool(mcplib.NewTool("environments",
		mcplib.WithDescription("Lists all APM environments available in the given time range."),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
	), environmentsHandler(c, log, now))

	s.AddTool(mcplib.NewTool("apm_indices",
		mcplib.WithDescription("Returns the configured APM index patterns (transaction, span, error, metric, etc.)."),
	), apmIndicesHandler(c, log))
}

func serviceListHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "service_list", err)
		}
		out, err := c.ServiceList(ctx, apm.ServiceListParams{
			Environment: optString(req, "environment"),
			Kuery:       optString(req, "kuery"),
			Start:       start,
			End:         end,
		})
		if err != nil {
			return toolErr(log, "service_list", err)
		}
		return toolJSON(out)
	}
}

func serviceMetricsHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		service, err := reqString(req, "service")
		if err != nil {
			return toolErr(log, "service_metrics", err)
		}
		metric, err := reqString(req, "metric")
		if err != nil {
			return toolErr(log, "service_metrics", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "service_metrics", err)
		}
		out, err := c.ServiceMetrics(ctx, apm.ServiceMetricsParams{
			Service:         service,
			Metric:          metric,
			Environment:     optString(req, "environment"),
			Kuery:           optString(req, "kuery"),
			Start:           start,
			End:             end,
			TransactionType: optString(req, "transaction_type"),
			TransactionName: optString(req, "transaction_name"),
			Offset:          optString(req, "offset"),
		})
		if err != nil {
			return toolErr(log, "service_metrics", err)
		}
		return toolJSON(out)
	}
}

func serviceDependenciesHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		service, err := reqString(req, "service")
		if err != nil {
			return toolErr(log, "service_dependencies", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "service_dependencies", err)
		}
		out, err := c.ServiceDependencies(ctx, apm.DependenciesParams{
			Service:     service,
			Environment: optString(req, "environment"),
			Start:       start,
			End:         end,
			Offset:      optString(req, "offset"),
		})
		if err != nil {
			return toolErr(log, "service_dependencies", err)
		}
		return toolJSON(out)
	}
}

func environmentsHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "environments", err)
		}
		out, err := c.Environments(ctx, start, end)
		if err != nil {
			return toolErr(log, "environments", err)
		}
		return toolJSON(out)
	}
}

func apmIndicesHandler(c APMClient, log zerolog.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		out, err := c.GetAPMIndices(ctx)
		if err != nil {
			return toolErr(log, "apm_indices", err)
		}
		return toolJSON(out)
	}
}
