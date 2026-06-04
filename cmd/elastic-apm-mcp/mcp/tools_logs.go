package mcp

import (
	"context"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

func registerLogTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger, now func() time.Time) {
	s.AddTool(mcplib.NewTool("logs_search",
		mcplib.WithDescription("Searches correlated log lines by trace.id, transaction.id, span.id, or KQL. Returns raw log documents for correlation."),
		mcplib.WithString("trace_id", mcplib.Description("Filter by trace.id")),
		mcplib.WithString("transaction_id", mcplib.Description("Filter by transaction.id")),
		mcplib.WithString("span_id", mcplib.Description("Filter by span.id")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithNumber("size", mcplib.Description("Max results (default 50)")),
	), logsSearchHandler(c, log, now))

	s.AddTool(mcplib.NewTool("es_search",
		mcplib.WithDescription("Executes a raw Elasticsearch query against any index. Returns _source documents. Use apm_indices to discover available index patterns."),
		mcplib.WithString("index", mcplib.Required(), mcplib.Description("Index pattern (e.g. traces-apm*)")),
		mcplib.WithObject("query", mcplib.Required(), mcplib.Description("Elasticsearch query body (JSON object)")),
	), esSearchHandler(c, log))
}

func logsSearchHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "logs_search", err)
		}
		out, err := c.LogsSearch(ctx, apm.LogsParams{
			TraceID:       optString(req, "trace_id"),
			TransactionID: optString(req, "transaction_id"),
			SpanID:        optString(req, "span_id"),
			Kuery:         optString(req, "kuery"),
			Start:         start,
			End:           end,
			Size:          optInt(req, "size"),
		})
		if err != nil {
			return toolErr(log, "logs_search", err)
		}
		return toolJSON(out)
	}
}

func esSearchHandler(c APMClient, log zerolog.Logger) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		index, err := reqString(req, "index")
		if err != nil {
			return toolErr(log, "es_search", err)
		}
		query, err := reqMap(req, "query")
		if err != nil {
			return toolErr(log, "es_search", err)
		}
		out, err := c.RawSearch(ctx, index, query)
		if err != nil {
			return toolErr(log, "es_search", err)
		}
		return toolJSON(out)
	}
}
