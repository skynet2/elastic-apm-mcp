package apm

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestESQL_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/search/esql", r.URL.Path)
		assert.Empty(t, r.Header.Get("Elastic-Api-Version"), "esql must not send the APM date version header")

		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "FROM traces-apm* | STATS c = COUNT(*) BY service.name", params["query"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rawResponse":{"columns":[{"name":"c","type":"long"},{"name":"service.name","type":"keyword"}],"values":[[5,"svc-a"],[2,"svc-b"]]}}`))
	})

	out, err := client.ESQL(context.Background(), "FROM traces-apm* | STATS c = COUNT(*) BY service.name")
	require.NoError(t, err)

	require.Len(t, out.Columns, 2)
	assert.Equal(t, "c", out.Columns[0].Name)
	assert.Equal(t, "long", out.Columns[0].Type)

	assert.Equal(t, 2, out.RowCount)
	require.Len(t, out.Rows, 2)
	assert.Equal(t, float64(5), out.Rows[0]["c"])
	assert.Equal(t, "svc-a", out.Rows[0]["service.name"])
	assert.Equal(t, "svc-b", out.Rows[1]["service.name"])
}

func TestESQL_Failure(t *testing.T) {
	t.Run("empty_query", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		_, err := client.ESQL(context.Background(), "   ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "query required")
	})

	t.Run("server_error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		})
		_, err := client.ESQL(context.Background(), "FROM traces-apm* | LIMIT 1")
		require.Error(t, err)
	})
}
