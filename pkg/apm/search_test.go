package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEseSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		assert.Empty(t, r.Header.Get("Elastic-Api-Version"), "ese must not send api version header")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "ApiKey test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "true", r.Header.Get("kbn-xsrf"))
		assert.Equal(t, "Kibana", r.Header.Get("x-elastic-internal-origin"))

		var reqBody map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))

		params, _ := reqBody["params"].(map[string]any)
		assert.Equal(t, "logs-apm*,logs-*", params["index"])

		body, _ := params["body"].(map[string]any)
		assert.NotNil(t, body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rawResponse": map[string]any{
				"hits": map[string]any{
					"total": 2,
					"hits": []any{
						map[string]any{"_source": map[string]any{"message": "log line 1", "trace.id": "abc"}},
						map[string]any{"_source": map[string]any{"message": "log line 2", "trace.id": "abc"}},
					},
				},
			},
			"isPartial": false,
			"isRunning": false,
		})
	}))
	t.Cleanup(srv.Close)

	client := New(Config{BaseURL: srv.URL, APIKey: "test-key", HTTPClient: srv.Client()})
	sources, err := client.eseSearch(context.Background(), "logs-apm*,logs-*", map[string]any{
		"size": 2,
		"query": map[string]any{
			"term": map[string]any{"trace.id": "abc"},
		},
	})
	require.NoError(t, err)
	require.Len(t, sources, 2)
	assert.Equal(t, "log line 1", sources[0]["message"])
	assert.Equal(t, "log line 2", sources[1]["message"])
}

func TestEseSearch_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	t.Cleanup(srv.Close)

	client := New(Config{BaseURL: srv.URL, APIKey: "test-key", HTTPClient: srv.Client()})
	_, err := client.eseSearch(context.Background(), "logs-apm*,logs-*", map[string]any{})
	require.Error(t, err)

	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr)
}
