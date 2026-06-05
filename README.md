# elastic-apm-mcp

An MCP (Model Context Protocol) stdio server that gives LLM agents structured access to Elastic APM. The agent can investigate distributed traces, error groups, latency trends, and correlated logs without human intermediation — navigating from a label or service name all the way to a root-cause stacktrace in a single conversation.

## Features

- **Distributed trace retrieval** — fetch the full waterfall, inline errors, and entry transaction for any trace ID
- **Error analytics** — list error groups with occurrence counts; fetch chained exceptions and stacktraces
- **Latency, throughput, and error-rate time series** — chart service health trends over any time window
- **Log correlation** — pull log lines tied to a specific trace, transaction, or span ID
- **Label-based trace search** — find traces by arbitrary custom labels (`labels.customer_id:"123"`)
- **Raw Elasticsearch escape hatch** — run any ES query against any index when built-in tools are insufficient

## How it works

The server talks to **Kibana** over HTTPS. It never connects to Elasticsearch directly and requires no port-forward or cluster credentials.

- Structured APM views (service list, transaction groups, trace waterfall, error groups) use Kibana's internal APM API at `/internal/apm/*`.
- Log correlation and label-based search use the Kibana Elasticsearch proxy at `/internal/search/ese`.
- Auth is a **Kibana API key** sent as `Authorization: ApiKey <key>`. Every request also carries `kbn-xsrf: true` and `Elastic-Api-Version: 2023-10-31` (APM endpoints) as required by Kibana.
- **Custom headers** (configured in `headers`) are forwarded on every request. Use this for reverse-proxy auth tokens such as Cloudflare Access (`CF-Access-Client-Id` / `CF-Access-Client-Secret`).

## Install

```bash
go install github.com/skynet2/elastic-apm-mcp/cmd/elastic-apm-mcp@latest
```

Or build locally:

```bash
make build
# produces bin/elastic-apm-mcp
```

## Configuration

### config.yaml

```yaml
url: https://kibana.example.com       # Kibana base URL (required)
api_key: your-base64-kibana-api-key   # Kibana API key (required)
headers:                              # Optional — forwarded on every request
  CF-Access-Client-Id: your-cf-client-id
  CF-Access-Client-Secret: your-cf-client-secret
timeout: 30s                          # HTTP timeout (default: 30s)
log_level: info                       # Zerolog level: trace|debug|info|warn|error
```

A ready-to-copy example lives at [`configs/config.example.yaml`](configs/config.example.yaml).

### Environment variables

All keys can be overridden with env vars prefixed `APM_`:

| Env var | Config key |
|---|---|
| `APM_URL` | `url` |
| `APM_API_KEY` | `api_key` |
| `APM_HEADERS` | `headers` |
| `APM_TIMEOUT` | `timeout` |
| `APM_LOG_LEVEL` | `log_level` |

`APM_HEADERS` is a comma-separated list of `Name=Value` pairs, e.g.
`APM_HEADERS=CF-Access-Client-Id=<id>,CF-Access-Client-Secret=<secret>`.
Env headers merge over (and override) any `headers` from the YAML file, so the
server can be configured entirely from the environment — useful for MCP client
`env` blocks.

### Creating a Kibana API key

1. Open Kibana → **Stack Management** → **API keys**.
2. Click **Create API key**.
3. Give it a name, set an expiry if required, and grant at least read access to APM and index patterns.
4. Copy the **Base64** encoded value and paste it into `api_key`.

Never commit real keys. Keep your config file out of version control (add it to `.gitignore`).

## MCP client setup

Add the server to your MCP client configuration (Claude Desktop, Claude Code, etc.):

```json
{
  "mcpServers": {
    "elastic-apm": {
      "command": "/path/to/bin/elastic-apm-mcp",
      "args": ["--config", "/path/to/config.yaml"]
    }
  }
}
```

Replace the paths with the actual binary location and your config file.

## Tools

Time arguments (`start`, `end`) accept ISO 8601 (`2026-01-15T10:00:00Z`) or relative strings (`now-15m`, `now-1h`, `now-1d`, `now`).

KQL `kuery` arguments support Kibana Query Language: `labels.<key>:"<value>"`, `service.name:"payment-service"`, `transaction.name:"POST /checkout"`, `trace.id:"abc123"`, etc.

---

### Services & metrics

| Tool | Purpose | Key params |
|---|---|---|
| `service_list` | List all APM services with latency, throughput, and error rate | `environment`, `kuery`, `start`, `end` |
| `service_metrics` | Latency / throughput / error_rate / breakdown time series for a service | `service`*, `metric`* (`latency\|throughput\|error_rate\|breakdown`), `environment`, `transaction_type`, `transaction_name`, `offset`, `start`, `end` |
| `service_dependencies` | Downstream dependencies with stats (dependency name, span type, error rate) | `service`*, `environment`, `offset`, `start`, `end` |
| `environments` | All APM environments in a time range | `start`, `end` |
| `apm_indices` | Configured APM index patterns (use with `es_search`) | — |

### Transactions

| Tool | Purpose | Key params |
|---|---|---|
| `transaction_groups` | Transaction groups for a service with latency, throughput, error rate, and impact | `service`*, `environment`, `kuery`, `transaction_type`, `start`, `end` |
| `transaction_samples` | Trace samples (trace.id + transaction.id) for a transaction name | `service`*, `environment`, `kuery`, `transaction_type`, `transaction_name`, `start`, `end` |

### Traces

| Tool | Purpose | Key params |
|---|---|---|
| `trace_get` | Full distributed trace waterfall with inline errors and entry transaction | `trace_id`*, `entry_transaction_id`, `start`, `end` |
| `trace_search` | Search transactions by KQL/label; returns trace.id + transaction.id | `kuery`, `service`, `size`, `start`, `end` |

### Errors

| Tool | Purpose | Key params |
|---|---|---|
| `error_groups` | Error groups for a service with occurrence counts and grouping keys | `service`*, `environment`, `kuery`, `start`, `end` |
| `error_get` | Full error detail: chained exceptions, stacktrace, and back-refs (trace.id, transaction.id, span.id) | `error_id`, `grouping_key` |

### Logs & raw search

| Tool | Purpose | Key params |
|---|---|---|
| `logs_search` | Log lines correlated to a trace/transaction/span or matching a KQL filter | `trace_id`, `transaction_id`, `span_id`, `kuery`, `size`, `start`, `end` |
| `es_search` | Raw Elasticsearch query against any index (escape hatch) | `index`*, `query`* (ES query body) |

\* Required parameter.

---

## Correlation model

Every tool returns the correlation IDs it touches so the agent can pivot in any direction:

```
labels.*                 ──trace_search──▶  trace.id + transaction.id
trace.id                 ──trace_get─────▶  waterfall + inline errors + entry transaction
error.id / grouping_key  ──error_get─────▶  exceptions + stacktrace + trace.id / transaction.id
trace.id / txn.id / span.id ──logs_search▶ correlated log lines
service + txn.name       ──error_groups──▶  error groups for that endpoint
service                  ──service_metrics▶ latency / throughput / error_rate time series
```

### Example: find a specific customer's failing request

```
1. trace_search(kuery: 'labels.customer_id:"cust-42"', start: "now-1h", end: "now")
   → returns trace.id, transaction.id

2. trace_get(trace_id: "<trace.id>", start: "now-1h", end: "now")
   → waterfall spans + inline errors[] with error.id

3. error_get(error_id: "<error.id>")
   → chained exception, full stacktrace, labels

4. logs_search(trace_id: "<trace.id>", start: "now-1h", end: "now")
   → correlated application log lines around the failure
```

**Common KQL patterns:**

```
labels.customer_id:"cust-42"
labels.order_id:"ord-999"
service.name:"payment-service"
transaction.name:"POST /checkout"
trace.id:"a1b2c3d4e5f6..."
processor.event:("error" OR "transaction")
```

## Development

```bash
make build      # compile → bin/elastic-apm-mcp
make test       # go test -p 1 -timeout 60s ./...
make test-e2e   # end-to-end tests (requires APM_URL + APM_API_KEY env vars)
make lint       # golangci-lint run
make generate   # regenerate mocks after interface changes (uses gomock)
make run        # go run ./cmd/elastic-apm-mcp
make tidy       # go mod tidy
make clean      # remove bin/
```

Tests use [`gomock`](https://github.com/golang/mock). After changing any interface, run `make generate` to regenerate mocks — stale mocks cause compilation errors.

## License

MIT — see [LICENSE](LICENSE).
