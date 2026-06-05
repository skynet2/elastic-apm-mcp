package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eseEnvelopeFromFixture(t *testing.T, fixturePath string) []byte {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	var source map[string]any
	require.NoError(t, json.Unmarshal(raw, &source))

	envelope := map[string]any{
		"rawResponse": map[string]any{
			"hits": map[string]any{
				"hits": []any{
					map[string]any{"_source": source},
				},
			},
		},
	}
	out, err := json.Marshal(envelope)
	require.NoError(t, err)
	return out
}

func decodeEseRequest(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	return body
}

func TestErrorGet_Success(t *testing.T) {
	errorDocEnvelope := eseEnvelopeFromFixture(t, "testdata/error_doc.json")

	t.Run("by errorId", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/internal/search/ese", r.URL.Path)
			body := decodeEseRequest(t, r)
			params := body["params"].(map[string]any)
			assert.Equal(t, "logs-apm*", params["index"])

			eseBody := params["body"].(map[string]any)
			filters := eseBody["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)
			assert.ElementsMatch(t, []any{
				map[string]any{"term": map[string]any{"error.id": "erraaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
			}, filters)
			assert.NotContains(t, eseBody, "sort", "errorId path must not set sort")
			assert.NotContains(t, eseBody, "size", "errorId path must not set size")

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(errorDocEnvelope)
		})

		results, err := client.ErrorGet(context.Background(), ErrorGetParams{ErrorID: "erraaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
		require.NoError(t, err)
		require.Len(t, results, 1)
		errDoc := results[0]["error"].(map[string]any)
		assert.Equal(t, "erraaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", errDoc["id"])
	})

	t.Run("by groupingKey", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			body := decodeEseRequest(t, r)
			params := body["params"].(map[string]any)
			eseBody := params["body"].(map[string]any)
			filters := eseBody["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)
			assert.ElementsMatch(t, []any{
				map[string]any{"term": map[string]any{"error.grouping_key": "grp0000000000001"}},
			}, filters)
			assert.Equal(t, []any{map[string]any{"@timestamp": "desc"}}, eseBody["sort"])
			assert.Equal(t, float64(1), eseBody["size"])

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(errorDocEnvelope)
		})

		results, err := client.ErrorGet(context.Background(), ErrorGetParams{GroupingKey: "grp0000000000001"})
		require.NoError(t, err)
		require.Len(t, results, 1)
		errDoc := results[0]["error"].(map[string]any)
		assert.Equal(t, "grp0000000000001", errDoc["grouping_key"])
	})
}

func TestErrorGet_Failure(t *testing.T) {
	t.Run("neither id nor groupingKey", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("no HTTP call expected")
		})

		_, err := client.ErrorGet(context.Background(), ErrorGetParams{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "errorId or groupingKey required")
	})

	t.Run("http 500", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		})

		_, err := client.ErrorGet(context.Background(), ErrorGetParams{ErrorID: "x"})
		require.Error(t, err)
	})
}

func TestLogsSearch_Success(t *testing.T) {
	logDocEnvelope := eseEnvelopeFromFixture(t, "testdata/log_doc.json")

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "logs-apm*,logs-*", params["index"])

		eseBody := params["body"].(map[string]any)
		filters := eseBody["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)
		assert.ElementsMatch(t, []any{
			map[string]any{"term": map[string]any{"trace.id": "traceaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
			map[string]any{"range": map[string]any{"@timestamp": map[string]any{
				"gte": "2024-01-01T00:00:00.000Z",
				"lte": "2024-01-02T00:00:00.000Z",
			}}},
		}, filters)
		assert.Equal(t, []any{map[string]any{"@timestamp": "desc"}}, eseBody["sort"])
		assert.Equal(t, float64(50), eseBody["size"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(logDocEnvelope)
	})

	results, err := client.LogsSearch(context.Background(), LogsParams{
		TraceID: "traceaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Start:   "2024-01-01T00:00:00.000Z",
		End:     "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "processing checkout", results[0]["message"])
}

func TestLogsSearch_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.LogsSearch(context.Background(), LogsParams{TraceID: "x"})
	require.Error(t, err)
}

func TestTraceSearch_Success(t *testing.T) {
	logDocEnvelope := eseEnvelopeFromFixture(t, "testdata/log_doc.json")

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "traces-apm*", params["index"])

		eseBody := params["body"].(map[string]any)
		filters := eseBody["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)
		assert.ElementsMatch(t, []any{
			map[string]any{"term": map[string]any{"processor.event": "transaction"}},
			map[string]any{"term": map[string]any{"service.name": "payment-service"}},
			map[string]any{"range": map[string]any{"@timestamp": map[string]any{
				"gte": "2024-01-01T00:00:00.000Z",
				"lte": "2024-01-02T00:00:00.000Z",
			}}},
		}, filters)
		assert.Equal(t, []any{map[string]any{"@timestamp": "desc"}}, eseBody["sort"])
		assert.Equal(t, float64(50), eseBody["size"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(logDocEnvelope)
	})

	results, err := client.TraceSearch(context.Background(), TraceSearchParams{
		Service: "payment-service",
		Start:   "2024-01-01T00:00:00.000Z",
		End:     "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	traceVal := results[0]["trace"].(map[string]any)
	assert.Equal(t, "traceaaaaaaaaaaaaaaaaaaaaaaaaaaaa", traceVal["id"])
}

func TestTraceSearch_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.TraceSearch(context.Background(), TraceSearchParams{})
	require.Error(t, err)
}

func TestRawSearch_Success(t *testing.T) {
	logDocEnvelope := eseEnvelopeFromFixture(t, "testdata/log_doc.json")

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "my-index*", params["index"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(logDocEnvelope)
	})

	results, err := client.RawSearch(context.Background(), "my-index*", map[string]any{"query": map[string]any{"match_all": map[string]any{}}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "processing checkout", results[0]["message"])
}

func TestRawSearch_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.RawSearch(context.Background(), "idx", map[string]any{})
	require.Error(t, err)
}
