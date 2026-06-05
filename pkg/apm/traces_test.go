package apm

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceGet_Success(t *testing.T) {
	data, err := os.ReadFile("testdata/trace.json")
	require.NoError(t, err)

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/unified_traces/traceaaaaaaaaaaaaaaaaaaaaaaaaaaaa", r.URL.Path)
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "txn0000000000001", r.URL.Query().Get("entryTransactionId"))
		assert.Equal(t, "true", r.URL.Query().Get("ecsOnly"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	trace, err := client.TraceGet(context.Background(), "traceaaaaaaaaaaaaaaaaaaaaaaaaaaaa", TraceParams{
		Start:              "2024-01-01T00:00:00.000Z",
		End:                "2024-01-02T00:00:00.000Z",
		EntryTransactionID: "txn0000000000001",
	})
	require.NoError(t, err)
	require.Len(t, trace.TraceItems, 1)
	assert.Equal(t, "payment-service", trace.TraceItems[0].ServiceName)
	require.Len(t, trace.Errors, 1)
	assert.Equal(t, "(*Service).Process", trace.Errors[0].Error.Culprit)
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
