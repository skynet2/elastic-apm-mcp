package mcp

import (
	"context"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

func registerErrorTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger, now func() time.Time) {
	s.AddTool(mcplib.NewTool("error_groups",
		mcplib.WithDescription("Lists error groups for a service with counts and grouping keys. Returns error.id and grouping_key for pivot into error_get."),
		mcplib.WithString("service", mcplib.Required(), mcplib.Description("Service name")),
		mcplib.WithString("environment", mcplib.Description("Filter by environment")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
	), errorGroupsHandler(c, log, now))

	s.AddTool(mcplib.NewTool("error_get",
		mcplib.WithDescription("Fetches full error details (chained exceptions, stacktrace, trace.id, transaction.id, span.id) by error_id or grouping_key. Use error.id/grouping_key from error_groups."),
		mcplib.WithString("error_id", mcplib.Description("Error document ID")),
		mcplib.WithString("grouping_key", mcplib.Description("Error grouping key")),
	), errorGetHandler(c, log))
}

func errorGroupsHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		service, err := reqString(req, "service")
		if err != nil {
			return toolErr(log, "error_groups", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "error_groups", err)
		}
		out, err := c.ErrorGroups(ctx, apm.ErrorGroupsParams{
			Service:     service,
			Environment: optString(req, "environment"),
			Kuery:       optString(req, "kuery"),
			Start:       start,
			End:         end,
		})
		if err != nil {
			return toolErr(log, "error_groups", err)
		}
		return toolJSON(out)
	}
}

func errorGetHandler(c APMClient, log zerolog.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		out, err := c.ErrorGet(ctx, apm.ErrorGetParams{
			ErrorID:     optString(req, "error_id"),
			GroupingKey: optString(req, "grouping_key"),
		})
		if err != nil {
			return toolErr(log, "error_get", err)
		}
		return toolJSON(out)
	}
}
