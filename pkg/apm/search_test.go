package apm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEseSearch_Success(t *testing.T) {
	envelope := eseEnvelopeFromFixture(t, "testdata/log_doc.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/search/ese", r.URL.Path)
		assert.Empty(t, r.Header.Get("Elastic-Api-Version"), "ese must not send api version header")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "ApiKey test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "true", r.Header.Get("kbn-xsrf"))
		assert.Equal(t, "Kibana", r.Header.Get("x-elastic-internal-origin"))

		body := decodeEseRequest(t, r)
		params, _ := body["params"].(map[string]any)
		assert.Equal(t, "logs-apm*,logs-*", params["index"])

		eseBody, _ := params["body"].(map[string]any)
		assert.NotNil(t, eseBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(envelope)
	}))
	t.Cleanup(srv.Close)

	client := New(Config{BaseURL: srv.URL, APIKey: "test-key", HTTPClient: srv.Client()})
	sources, err := client.eseSearch(context.Background(), "logs-apm*,logs-*", map[string]any{
		"size": 1,
		"query": map[string]any{
			"term": map[string]any{"trace.id": "traceaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
	})
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, "processing checkout", sources[0]["message"])
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
