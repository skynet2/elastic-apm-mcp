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
- trace.id/transaction.id/span.id --logs_search--> correlated log lines
- service + transaction.name --error_groups--> errors for that endpoint
- service --service_metrics--> latency/throughput/error_rate/breakdown time series
Every tool returns correlation ids so you can pivot in any direction.
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
	return s
}
