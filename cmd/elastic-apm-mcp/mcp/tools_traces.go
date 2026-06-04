package mcp

import (
	"context"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

func registerTraceTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger, now func() time.Time) {
	s.AddTool(mcplib.NewTool("trace_get",
		mcplib.WithDescription("Returns the full distributed trace waterfall, inline errors, and entry transaction with labels — use trace.id from trace_search/error_get/transaction_samples."),
		mcplib.WithString("trace_id", mcplib.Required(), mcplib.Description("Trace ID")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithString("entry_transaction_id", mcplib.Description("Entry transaction ID hint")),
	), traceGetHandler(c, log, now))

	s.AddTool(mcplib.NewTool("trace_search",
		mcplib.WithDescription("Searches transactions by KQL/service and returns trace.id + transaction.id for pivot into trace_get or logs_search."),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("service", mcplib.Description("Service name filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithNumber("size", mcplib.Description("Max results (default 50)")),
	), traceSearchHandler(c, log, now))
}

func traceGetHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		traceID, err := reqString(req, "trace_id")
		if err != nil {
			return toolErr(log, "trace_get", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "trace_get", err)
		}
		out, err := c.TraceGet(ctx, traceID, apm.TraceParams{
			Start:              start,
			End:                end,
			EntryTransactionID: optString(req, "entry_transaction_id"),
		})
		if err != nil {
			return toolErr(log, "trace_get", err)
		}
		return toolJSON(out)
	}
}

func traceSearchHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "trace_search", err)
		}
		out, err := c.TraceSearch(ctx, apm.TraceSearchParams{
			Kuery:   optString(req, "kuery"),
			Service: optString(req, "service"),
			Start:   start,
			End:     end,
			Size:    optInt(req, "size"),
		})
		if err != nil {
			return toolErr(log, "trace_search", err)
		}
		return toolJSON(out)
	}
}
