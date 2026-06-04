package mcp

import (
	"context"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

func registerTransactionTools(s *mcpserver.MCPServer, c APMClient, log zerolog.Logger, now func() time.Time) {
	s.AddTool(mcplib.NewTool("transaction_groups",
		mcplib.WithDescription("Lists transaction groups for a service with latency, throughput, error rate, and impact. Returns transaction names for further pivot via transaction_samples."),
		mcplib.WithString("service", mcplib.Required(), mcplib.Description("Service name")),
		mcplib.WithString("environment", mcplib.Description("Filter by environment")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithString("transaction_type", mcplib.Description("Transaction type (default: request)")),
	), transactionGroupsHandler(c, log, now))

	s.AddTool(mcplib.NewTool("transaction_samples",
		mcplib.WithDescription("Returns trace samples (trace.id + transaction.id) for a transaction. Use trace.id with trace_get to fetch the full waterfall."),
		mcplib.WithString("service", mcplib.Required(), mcplib.Description("Service name")),
		mcplib.WithString("environment", mcplib.Description("Filter by environment")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithString("transaction_type", mcplib.Description("Transaction type (default: request)")),
		mcplib.WithString("transaction_name", mcplib.Description("Transaction name filter")),
	), transactionSamplesHandler(c, log, now))
}

func transactionGroupsHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		service, err := reqString(req, "service")
		if err != nil {
			return toolErr(log, "transaction_groups", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "transaction_groups", err)
		}
		out, err := c.TransactionGroups(ctx, apm.TransactionGroupsParams{
			Service:         service,
			Environment:     optString(req, "environment"),
			Kuery:           optString(req, "kuery"),
			Start:           start,
			End:             end,
			TransactionType: optString(req, "transaction_type"),
		})
		if err != nil {
			return toolErr(log, "transaction_groups", err)
		}
		return toolJSON(out)
	}
}

func transactionSamplesHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		service, err := reqString(req, "service")
		if err != nil {
			return toolErr(log, "transaction_samples", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "transaction_samples", err)
		}
		out, err := c.TransactionSamples(ctx, apm.TransactionSamplesParams{
			Service:         service,
			Environment:     optString(req, "environment"),
			Kuery:           optString(req, "kuery"),
			Start:           start,
			End:             end,
			TransactionType: optString(req, "transaction_type"),
			TransactionName: optString(req, "transaction_name"),
		})
		if err != nil {
			return toolErr(log, "transaction_samples", err)
		}
		return toolJSON(out)
	}
}
