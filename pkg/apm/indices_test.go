package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func aggEnvelope(t *testing.T, keys ...string) []byte {
	t.Helper()
	buckets := make([]any, 0, len(keys))
	for _, k := range keys {
		buckets = append(buckets, map[string]any{"key": k, "doc_count": 1})
	}
	envelope := map[string]any{
		"rawResponse": map[string]any{
			"hits":         map[string]any{"hits": []any{}},
			"aggregations": map[string]any{"indices": map[string]any{"buckets": buckets}},
		},
	}
	out, err := json.Marshal(envelope)
	require.NoError(t, err)
	return out
}

func TestListIndices_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/search/ese", r.URL.Path)

		body := decodeEseRequest(t, r)
		params := body["params"].(map[string]any)
		assert.Equal(t, "*", params["index"])

		eseBody := params["body"].(map[string]any)
		assert.Equal(t, float64(0), eseBody["size"])
		terms := eseBody["aggs"].(map[string]any)["indices"].(map[string]any)["terms"].(map[string]any)
		assert.Equal(t, "_index", terms["field"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(aggEnvelope(t, "index-b", "index-a"))
	})

	out, err := client.ListIndices(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, []string{"index-a", "index-b"}, out)
}

func TestListIndices_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.ListIndices(context.Background(), "app-logs-*")
	require.Error(t, err)
}

func TestDescribeFields_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/internal/data_views/fields", r.URL.Path)
		assert.Empty(t, r.Header.Get("Elastic-Api-Version"), "data_views must not send the APM date version header")

		q := r.URL.Query()
		assert.Equal(t, "app-logs-*", q.Get("pattern"))
		assert.Equal(t, "1", q.Get("apiVersion"))
		assert.Equal(t, "true", q.Get("allow_no_index"))
		assert.Contains(t, q["meta_fields"], "_index")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"fields":[{"name":"service_name","type":"string","esTypes":["text"],"searchable":true,"aggregatable":false},{"name":"trace_id","type":"string","esTypes":["keyword"],"searchable":true,"aggregatable":true}]}`))
	})

	out, err := client.DescribeFields(context.Background(), "app-logs-*")
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "service_name", out[0].Name)
	assert.Equal(t, []string{"text"}, out[0].ESTypes)
	assert.Equal(t, []string{"keyword"}, out[1].ESTypes)
}

func TestDescribeFields_Failure(t *testing.T) {
	t.Run("missing_pattern", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		_, err := client.DescribeFields(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern required")
	})

	t.Run("server_error", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		})
		_, err := client.DescribeFields(context.Background(), "app-logs-*")
		require.Error(t, err)
	})
}

func TestEseSearchResult_IndexAndAggregations(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rawResponse":{"hits":{"total":{"value":2},"hits":[{"_index":"app-logs-000001","_source":{"message":"hi"}}]},"aggregations":{"levels":{"buckets":[{"key":"info"}]}}}}`))
	})

	res, err := client.eseSearchResult(context.Background(), "app-logs-*", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Total)
	require.Len(t, res.Hits, 1)
	assert.Equal(t, "app-logs-000001", res.Hits[0]["_index"])
	assert.Equal(t, "hi", res.Hits[0]["message"])
	assert.NotNil(t, res.Aggregations["levels"])
}

func TestEseSearchResult_TotalShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{
			name: "number_total",
			body: `{"rawResponse":{"hits":{"total":42,"hits":[]}}}`,
			want: 42,
		},
		{
			name: "object_total",
			body: `{"rawResponse":{"hits":{"total":{"value":7,"relation":"eq"},"hits":[]}}}`,
			want: 7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			})

			res, err := client.eseSearchResult(context.Background(), "app-logs-*", map[string]any{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, res.Total)
		})
	}
}
