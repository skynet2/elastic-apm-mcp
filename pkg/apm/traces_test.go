package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceGet_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/unified_traces/trace-abc", r.URL.Path)
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "tx-123", r.URL.Query().Get("entryTransactionId"))
		assert.Equal(t, "true", r.URL.Query().Get("ecsOnly"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"traceItems": []any{
				map[string]any{
					"id":                 "item-1",
					"parentId":           "",
					"name":               "GET /api",
					"serviceName":        "my-svc",
					"serviceEnvironment": "production",
					"agentName":          "go",
					"docType":            "transaction",
					"duration":           int64(1500),
					"result":             "success",
					"timestampUs":        int64(1704067200000000),
					"traceId":            "trace-abc",
					"errors":             []any{},
				},
			},
			"errors": []any{
				map[string]any{
					"id":    "err-1",
					"index": "logs-apm.error",
					"error": map[string]any{
						"culprit":      "handler.go:42",
						"grouping_key": "gkey-1",
						"id":           "err-1",
						"exception": map[string]any{
							"handled": false,
							"message": "nil pointer dereference",
							"type":    "runtime error",
						},
					},
					"service":     map[string]any{"name": "my-svc"},
					"transaction": map[string]any{"id": "tx-123"},
					"span":        map[string]any{"id": ""},
					"trace":       map[string]any{"id": "trace-abc"},
					"parent":      map[string]any{"id": "tx-123"},
					"timestamp":   map[string]any{"us": int64(1704067200000000)},
				},
			},
			"entryTransaction": map[string]any{"service": map[string]any{"name": "my-svc"}},
			"traceDocsTotal":   1,
			"maxTraceItems":    1,
		})
	})

	trace, err := client.TraceGet(context.Background(), "trace-abc", TraceParams{
		Start:              "2024-01-01T00:00:00.000Z",
		End:                "2024-01-02T00:00:00.000Z",
		EntryTransactionID: "tx-123",
	})
	require.NoError(t, err)
	require.Len(t, trace.TraceItems, 1)
	assert.Equal(t, "my-svc", trace.TraceItems[0].ServiceName)
	require.Len(t, trace.Errors, 1)
	assert.Equal(t, "handler.go:42", trace.Errors[0].Error.Culprit)
	assert.NotNil(t, trace.EntryTransaction)
	assert.Equal(t, 1, trace.TraceDocsTotal)
}

func TestTraceGet_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.TraceGet(context.Background(), "missing", TraceParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trace get")
}
