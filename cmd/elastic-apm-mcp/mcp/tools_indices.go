package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
)

func registerIndexTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger) {
	s.AddTool(mcplib.NewTool("list_indices",
		mcplib.WithDescription("Lists concrete Elasticsearch index names matching a pattern (default *). Use this to discover where data lives - e.g. that application logs are under fluent-bit-* - instead of guessing patterns."),
		mcplib.WithString("pattern", mcplib.Description("Index pattern to enumerate (default *)")),
	), listIndicesHandler(c, log))

	s.AddTool(mcplib.NewTool("describe_fields",
		mcplib.WithDescription("Lists the fields of an index pattern with their Elasticsearch types (keyword vs text, aggregatable, searchable). Use it to decide whether a field needs a term filter (keyword) or match/match_phrase (text) - term silently misses on analyzed fields like service_name / k8_pod_name."),
		mcplib.WithString("pattern", mcplib.Required(), mcplib.Description("Index pattern (e.g. fluent-bit-*)")),
	), describeFieldsHandler(c, log))
}

func listIndicesHandler(c APMClient, log zerolog.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		out, err := c.ListIndices(ctx, optString(req, "pattern"))
		if err != nil {
			return toolErr(log, "list_indices", err)
		}
		return toolJSON(out)
	}
}

func describeFieldsHandler(c APMClient, log zerolog.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		pattern, err := reqString(req, "pattern")
		if err != nil {
			return toolErr(log, "describe_fields", err)
		}
		out, err := c.DescribeFields(ctx, pattern)
		if err != nil {
			return toolErr(log, "describe_fields", err)
		}
		return toolJSON(out)
	}
}
