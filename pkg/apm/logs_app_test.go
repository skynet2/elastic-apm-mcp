package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eseEnvelope(t *testing.T, sources ...map[string]any) []byte {
	t.Helper()
	hits := make([]any, 0, len(sources))
	for _, s := range sources {
		hits = append(hits, map[string]any{"_source": s})
	}
	envelope := map[string]any{
		"rawResponse": map[string]any{
			"hits": map[string]any{"hits": hits},
		},
	}
	out, err := json.Marshal(envelope)
	require.NoError(t, err)
	return out
}

func TestAppLogsSearch_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)

		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, defaultAppLogsIndex, params["index"])

		eseBody := params["body"].(map[string]any)
		filters := eseBody["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)
		assert.Contains(t, filters, map[string]any{"match_phrase": map[string]any{"service_name": "payment-service"}})
		assert.Contains(t, filters, map[string]any{"match_phrase": map[string]any{"k8_pod_name": "payment-service-1"}})
		assert.Contains(t, filters, map[string]any{"term": map[string]any{"trace_id": "trace-1"}})
		assert.Contains(t, filters, map[string]any{"match_phrase": map[string]any{"message": "processing request"}})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(eseEnvelope(t, map[string]any{"response_body": `{"status":"ok"}`}))
	})

	out, err := client.AppLogsSearch(context.Background(), AppLogsParams{
		Service: "payment-service",
		Pod:     "payment-service-1",
		TraceID: "trace-1",
		Message: "processing request",
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, `{"status":"ok"}`, out[0]["response_body"])
}

func TestAppLogsSearch_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.AppLogsSearch(context.Background(), AppLogsParams{Service: "payment-service"})
	require.Error(t, err)

	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr)
}

func TestTraceLogs_Success(t *testing.T) {
	var indices []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		indices = append(indices, params["index"].(string))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(eseEnvelope(t, map[string]any{"message": "m", "@timestamp": "2026-01-15T10:00:00Z"}))
	})

	out, err := client.TraceLogs(context.Background(), TraceLogsParams{TraceID: "trace-1"})
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Contains(t, indices, "logs-apm*")
	assert.Contains(t, indices, defaultAppLogsIndex)
}

func TestTraceLogs_Failure(t *testing.T) {
	t.Run("missing_trace_id", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		_, err := client.TraceLogs(context.Background(), TraceLogsParams{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "traceId required")
	})

	t.Run("server_error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		})
		_, err := client.TraceLogs(context.Background(), TraceLogsParams{TraceID: "trace-1"})
		require.Error(t, err)
	})
}

func TestSortByTimestampDesc(t *testing.T) {
	docs := []map[string]any{
		{"id": "a", "@timestamp": "2026-01-15T10:00:00Z"},
		{"id": "b", "@timestamp": "2026-01-15T10:05:00Z"},
		{"id": "c"},
		{"id": "d", "@timestamp": "2026-01-15T10:02:00Z"},
	}

	sortByTimestampDesc(docs)

	got := []string{docs[0]["id"].(string), docs[1]["id"].(string), docs[2]["id"].(string), docs[3]["id"].(string)}
	assert.Equal(t, []string{"b", "d", "a", "c"}, got)
}
