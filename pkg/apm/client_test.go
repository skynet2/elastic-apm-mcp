package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(Config{
		BaseURL:    srv.URL,
		APIKey:     "test-key",
		HTTPClient: srv.Client(),
	})
}

func newTestClientWithHeaders(t *testing.T, handler http.HandlerFunc, hdrs map[string]string) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(Config{
		BaseURL:    srv.URL,
		APIKey:     "test-key",
		Headers:    hdrs,
		HTTPClient: srv.Client(),
	})
}

func TestClient_Headers(t *testing.T) {
	var capturedReq *http.Request
	client := newTestClientWithHeaders(t, func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}, map[string]string{
		"CF-Access-Client-Id": "cid",
	})

	var out map[string]any
	err := client.get(context.Background(), "/internal/apm/test", nil, &out)
	require.NoError(t, err)

	assert.Equal(t, "ApiKey test-key", capturedReq.Header.Get("Authorization"))
	assert.Equal(t, "application/json", capturedReq.Header.Get("Accept"))
	assert.Equal(t, "true", capturedReq.Header.Get("kbn-xsrf"))
	assert.Equal(t, "Kibana", capturedReq.Header.Get("x-elastic-internal-origin"))
	assert.Equal(t, "cid", capturedReq.Header.Get("CF-Access-Client-Id"))
}

func TestClient_ApmVersionHeader(t *testing.T) {
	tests := []struct {
		name   string
		invoke func(c *Client, srv *httptest.Server) error
	}{
		{
			name: "get carries version",
			invoke: func(c *Client, _ *httptest.Server) error {
				var out map[string]any
				return c.get(context.Background(), "/internal/apm/test", nil, &out)
			},
		},
		{
			name: "post carries version",
			invoke: func(c *Client, _ *httptest.Server) error {
				var out map[string]any
				return c.post(context.Background(), "/internal/apm/test", nil, map[string]any{}, &out)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "2023-10-31", r.Header.Get("Elastic-Api-Version"))
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{})
			}))
			t.Cleanup(srv.Close)

			c := New(Config{BaseURL: srv.URL, APIKey: "test-key", HTTPClient: srv.Client()})
			require.NoError(t, tc.invoke(c, srv))
		})
	}
}

func TestClient_ErrorHandling_Success(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{name: "401 unauthorized", statusCode: http.StatusUnauthorized, wantErr: ErrUnauthorized},
		{name: "404 not found", statusCode: http.StatusNotFound, wantErr: ErrNotFound},
		{name: "403 forbidden", statusCode: http.StatusForbidden, wantErr: ErrForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			var out map[string]any
			err := client.get(context.Background(), "/internal/apm/test", nil, &out)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestClient_ErrorHandling_APIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	})

	var out map[string]any
	err := client.get(context.Background(), "/internal/apm/test", nil, &out)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	assert.Equal(t, "internal server error", apiErr.Message)
}

func TestClient_NilHTTPClient(t *testing.T) {
	c := New(Config{BaseURL: "http://localhost", APIKey: "key"})
	assert.Equal(t, 30*time.Second, c.httpClient.Timeout)
}

func TestClient_BaseURLTrailingSlash(t *testing.T) {
	c := New(Config{BaseURL: "http://localhost/", APIKey: "key"})
	assert.Equal(t, "http://localhost", c.baseURL)
}
