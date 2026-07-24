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
		mcplib.WithDescription("Searches APM-correlated log lines by trace.id, transaction.id, span.id, or KQL (index logs-apm*,logs-*). For application/container stdout logs use app_logs_search instead."),
		mcplib.WithString("trace_id", mcplib.Description("Filter by trace.id")),
		mcplib.WithString("transaction_id", mcplib.Description("Filter by transaction.id")),
		mcplib.WithString("span_id", mcplib.Description("Filter by span.id")),
		mcplib.WithString("kuery", mcplib.Description("KQL filter")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithNumber("size", mcplib.Description("Max results (default 50)")),
	), logsSearchHandler(c, log, now))

	s.AddTool(mcplib.NewTool("app_logs_search",
		mcplib.WithDescription("Searches application/container stdout logs (fluent-bit stream, flattened non-ECS schema: service_name, k8_pod_name, level, message, trace_id, transaction_id, plus structured fields like response_body at the top level). Use this to read what a service actually logged - the APM error index (logs_search) only holds captured errors, not Info/Warn lines. Filter by service/pod/trace_id, or drop into kuery for arbitrary fields (e.g. response_body:*need_auth*)."),
		mcplib.WithString("service", mcplib.Description("Service name (matched against service_name)")),
		mcplib.WithString("pod", mcplib.Description("Pod name or substring (matched against k8_pod_name)")),
		mcplib.WithString("namespace", mcplib.Description("Kubernetes namespace (k8_namespace)")),
		mcplib.WithString("container", mcplib.Description("Container name (k8_container_name)")),
		mcplib.WithString("trace_id", mcplib.Description("Correlate by flattened trace_id (keyword)")),
		mcplib.WithString("transaction_id", mcplib.Description("Correlate by flattened transaction_id (keyword)")),
		mcplib.WithString("message", mcplib.Description("Match the log message (e.g. \"GetItemByHash response\")")),
		mcplib.WithString("level", mcplib.Description("Log level: info, warn, error")),
		mcplib.WithString("kuery", mcplib.Description("KQL / query_string over any flattened field (e.g. market_name:ShadowPay)")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithNumber("size", mcplib.Description("Max results (default 50)")),
	), appLogsSearchHandler(c, log, now))

	s.AddTool(mcplib.NewTool("trace_logs",
		mcplib.WithDescription("Returns the full log timeline for a trace in one call: APM error docs plus fluent-bit application logs, merged and sorted newest-first. Each entry keeps its _index so you can tell APM errors from app logs."),
		mcplib.WithString("trace_id", mcplib.Required(), mcplib.Description("Trace id to gather logs for")),
		mcplib.WithString("start", mcplib.Description("Start time")),
		mcplib.WithString("end", mcplib.Description("End time")),
		mcplib.WithNumber("size", mcplib.Description("Max merged results (default 100)")),
	), traceLogsHandler(c, log, now))

	s.AddTool(mcplib.NewTool("es_search",
		mcplib.WithDescription("Executes a raw Elasticsearch query against any index. Returns {total, hits (each with _index), aggregations}. Use apm_indices or list_indices to discover index patterns."),
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

func appLogsSearchHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "app_logs_search", err)
		}
		out, err := c.AppLogsSearch(ctx, apm.AppLogsParams{
			Service:       optString(req, "service"),
			Pod:           optString(req, "pod"),
			Namespace:     optString(req, "namespace"),
			Container:     optString(req, "container"),
			TraceID:       optString(req, "trace_id"),
			TransactionID: optString(req, "transaction_id"),
			Message:       optString(req, "message"),
			Level:         optString(req, "level"),
			Kuery:         optString(req, "kuery"),
			Start:         start,
			End:           end,
			Size:          optInt(req, "size"),
		})
		if err != nil {
			return toolErr(log, "app_logs_search", err)
		}
		return toolJSON(out)
	}
}

func traceLogsHandler(c APMClient, log zerolog.Logger, now func() time.Time) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		traceID, err := reqString(req, "trace_id")
		if err != nil {
			return toolErr(log, "trace_logs", err)
		}
		start, end, err := resolveRange(req, now)
		if err != nil {
			return toolErr(log, "trace_logs", err)
		}
		out, err := c.TraceLogs(ctx, apm.TraceLogsParams{
			TraceID: traceID,
			Start:   start,
			End:     end,
			Size:    optInt(req, "size"),
		})
		if err != nil {
			return toolErr(log, "trace_logs", err)
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
