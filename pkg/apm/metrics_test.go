package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceMetrics_Success(t *testing.T) {
	tests := []struct {
		name     string
		metric   string
		wantPath string
	}{
		{
			name:     "latency",
			metric:   "latency",
			wantPath: "/internal/apm/services/my-svc/transactions/charts/latency",
		},
		{
			name:     "throughput",
			metric:   "throughput",
			wantPath: "/internal/apm/services/my-svc/throughput",
		},
		{
			name:     "error_rate",
			metric:   "error_rate",
			wantPath: "/internal/apm/services/my-svc/transactions/charts/error_rate",
		},
		{
			name:     "breakdown",
			metric:   "breakdown",
			wantPath: "/internal/apm/services/my-svc/transaction/charts/breakdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantPath, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{})
			})

			result, err := client.ServiceMetrics(context.Background(), ServiceMetricsParams{
				Service:         "my-svc",
				Metric:          tt.metric,
				Environment:     "production",
				Start:           "2024-01-01T00:00:00.000Z",
				End:             "2024-01-02T00:00:00.000Z",
				TransactionType: "request",
			})
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestServiceMetrics_Failure(t *testing.T) {
	t.Run("unsupported metric", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("no HTTP call expected for unsupported metric")
		})

		_, err := client.ServiceMetrics(context.Background(), ServiceMetricsParams{
			Service: "my-svc",
			Metric:  "unknown",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported metric")
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("http 500", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		})

		_, err := client.ServiceMetrics(context.Background(), ServiceMetricsParams{
			Service: "my-svc",
			Metric:  "latency",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service metrics")
	})
}
