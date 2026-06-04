package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cannedEseResponse(w http.ResponseWriter, source map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"rawResponse": map[string]any{
			"hits": map[string]any{
				"hits": []any{
					map[string]any{"_source": source},
				},
			},
		},
	})
}

func decodeEseRequest(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	return body
}

func TestErrorGet_Success(t *testing.T) {
	t.Run("by errorId", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/internal/search/ese", r.URL.Path)
			body := decodeEseRequest(t, r)
			params := body["params"].(map[string]any)
			assert.Equal(t, "logs-apm*", params["index"])

			eseBody := params["body"].(map[string]any)
			query := eseBody["query"].(map[string]any)
			boolQ := query["bool"].(map[string]any)
			filters := boolQ["filter"].([]any)
			found := false
			for _, f := range filters {
				fm := f.(map[string]any)
				if term, ok := fm["term"].(map[string]any); ok {
					if term["error.id"] == "err-123" {
						found = true
					}
				}
			}
			assert.True(t, found, "filter for error.id must exist")
			assert.NotContains(t, eseBody, "sort", "errorId path must not set sort")
			assert.NotContains(t, eseBody, "size", "errorId path must not set size")

			cannedEseResponse(w, map[string]any{"error": map[string]any{"id": "err-123"}})
		})

		results, err := client.ErrorGet(context.Background(), ErrorGetParams{ErrorID: "err-123"})
		require.NoError(t, err)
		require.Len(t, results, 1)
		errDoc := results[0]["error"].(map[string]any)
		assert.Equal(t, "err-123", errDoc["id"])
	})

	t.Run("by groupingKey", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			body := decodeEseRequest(t, r)
			params := body["params"].(map[string]any)
			eseBody := params["body"].(map[string]any)
			query := eseBody["query"].(map[string]any)
			boolQ := query["bool"].(map[string]any)
			filters := boolQ["filter"].([]any)
			found := false
			for _, f := range filters {
				fm := f.(map[string]any)
				if term, ok := fm["term"].(map[string]any); ok {
					if term["error.grouping_key"] == "gkey-1" {
						found = true
					}
				}
			}
			assert.True(t, found, "filter for error.grouping_key must exist")
			cannedEseResponse(w, map[string]any{"error": map[string]any{"grouping_key": "gkey-1"}})
		})

		results, err := client.ErrorGet(context.Background(), ErrorGetParams{GroupingKey: "gkey-1"})
		require.NoError(t, err)
		require.Len(t, results, 1)
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
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "logs-apm*,logs-*", params["index"])

		eseBody := params["body"].(map[string]any)
		query := eseBody["query"].(map[string]any)
		boolQ := query["bool"].(map[string]any)
		filters := boolQ["filter"].([]any)

		foundTrace := false
		for _, f := range filters {
			fm := f.(map[string]any)
			if term, ok := fm["term"].(map[string]any); ok {
				if term["trace.id"] == "trace-abc" {
					foundTrace = true
				}
			}
		}
		assert.True(t, foundTrace, "trace.id filter must exist")

		cannedEseResponse(w, map[string]any{"trace": map[string]any{"id": "trace-abc"}})
	})

	results, err := client.LogsSearch(context.Background(), LogsParams{
		TraceID: "trace-abc",
		Start:   "2024-01-01T00:00:00.000Z",
		End:     "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
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
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "traces-apm*", params["index"])

		eseBody := params["body"].(map[string]any)
		query := eseBody["query"].(map[string]any)
		boolQ := query["bool"].(map[string]any)
		filters := boolQ["filter"].([]any)

		foundEvent := false
		foundSvc := false
		for _, f := range filters {
			fm := f.(map[string]any)
			if term, ok := fm["term"].(map[string]any); ok {
				if term["processor.event"] == "transaction" {
					foundEvent = true
				}
				if term["service.name"] == "my-svc" {
					foundSvc = true
				}
			}
		}
		assert.True(t, foundEvent, "processor.event filter must exist")
		assert.True(t, foundSvc, "service.name filter must exist")

		cannedEseResponse(w, map[string]any{"trace": map[string]any{"id": "t1"}})
	})

	results, err := client.TraceSearch(context.Background(), TraceSearchParams{
		Service: "my-svc",
		Start:   "2024-01-01T00:00:00.000Z",
		End:     "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
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
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "my-index*", params["index"])

		cannedEseResponse(w, map[string]any{"field": "value"})
	})

	results, err := client.RawSearch(context.Background(), "my-index*", map[string]any{"query": map[string]any{"match_all": map[string]any{}}})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "value", results[0]["field"])
}

func TestRawSearch_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.RawSearch(context.Background(), "idx", map[string]any{})
	require.Error(t, err)
}
