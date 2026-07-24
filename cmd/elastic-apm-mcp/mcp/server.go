package mcp

import (
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
)

const serverInstructions = `This server investigates Elastic APM. Pivot graph:
- labels.* --trace_search--> trace.id + transaction.id
- trace.id --trace_get--> waterfall (traceItems) + errors[] inline + entryTransaction (full doc incl labels)
- error.id/grouping_key --error_get--> chained exceptions + stacktrace + back-refs (trace.id, transaction.id, span.id)
- trace.id/transaction.id/span.id --logs_search--> APM-correlated log lines (captured errors only)
- trace.id --trace_logs--> full timeline: APM errors + application logs merged
- service + transaction.name --error_groups--> errors for that endpoint
- service --service_metrics--> latency/throughput/error_rate/breakdown time series
Every tool returns correlation ids so you can pivot in any direction.
Application/container stdout logs are a SEPARATE stream from APM errors: index fluent-bit-* (configurable), flattened non-ECS schema. Use app_logs_search, NOT logs_search, to read what a service actually logged (Info/Warn lines, response bodies). Its fields: service_name, k8_pod_name, k8_namespace, level, message, trace_id, transaction_id (flattened - NOT trace.id/service.name), plus arbitrary structured fields (e.g. response_body) promoted to the top level. Correlate to APM by the flat trace_id/transaction_id. Text fields (service_name, k8_pod_name) are analyzed - a term query silently misses; app_logs_search handles this for you, and describe_fields tells you a field's type. list_indices discovers index names when you do not know them; es_search returns _index per hit plus aggregations for trend/terms analysis. esql runs raw ES|QL pipelines (FROM ... | WHERE ... | STATS ... BY) for cross-dataset aggregation that Query DSL cannot express in one call - scope the time range inside the query.
Kuery (KQL) for filtering: labels.<key>:"<val>", service.name:"x", transaction.name:"x", trace.id:"x", transaction.id:"x", span.id:"x", processor.event:("transaction" or "span" or "error").
Time args (start/end) accept ISO8601 or relative (now, now-15m, now-1d); default start=now-15m end=now.`

// NewServer builds and returns a configured MCPServer with all APM tools registered.
func NewServer(client APMClient, log zerolog.Logger, now func() time.Time, version string) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer("elastic-apm-mcp", version, mcpserver.WithInstructions(serverInstructions))
	registerServiceTools(s, client, log, now)
	registerTransactionTools(s, client, log, now)
	registerTraceTools(s, client, log, now)
	registerErrorTools(s, client, log, now)
	registerLogTools(s, client, log, now)
	registerIndexTools(s, client, log)
	registerESQLTools(s, client, log)
	return s
}
