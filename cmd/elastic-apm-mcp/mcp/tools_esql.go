package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
)

func registerESQLTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger) {
	s.AddTool(mcplib.NewTool("esql",
		mcplib.WithDescription("Runs a raw ES|QL pipeline query (FROM ... | WHERE ... | STATS ... | SORT ... | LIMIT). Use this for cross-dataset aggregation that Query DSL / es_search cannot express in one call, e.g. `FROM traces-apm*,apm-* | WHERE @timestamp > NOW() - 1 hour | STATS count = COUNT(*) BY service.name | SORT count DESC`. Returns {columns, rows, row_count}. Scope the time range inside the query; unbounded scans can time out."),
		mcplib.WithString("query", mcplib.Required(), mcplib.Description("ES|QL query string")),
	), esqlHandler(c, log))
}

func esqlHandler(c APMClient, log zerolog.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		query, err := reqString(req, "query")
		if err != nil {
			return toolErr(log, "esql", err)
		}
		out, err := c.ESQL(ctx, query)
		if err != nil {
			return toolErr(log, "esql", err)
		}
		return toolJSON(out)
	}
}
