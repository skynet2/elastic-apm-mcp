package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionGroups_Success(t *testing.T) {
	tests := []struct {
		name       string
		params     TransactionGroupsParams
		wantPath   string
		wantEnv    string
		wantTxType string
		wantTxName string
	}{
		{
			name: "explicit params",
			params: TransactionGroupsParams{
				Service:         "my-svc",
				Environment:     "production",
				Kuery:           "",
				Start:           "2024-01-01T00:00:00.000Z",
				End:             "2024-01-02T00:00:00.000Z",
				TransactionType: "request",
			},
			wantPath:   "/internal/apm/services/my-svc/transactions/groups/main_statistics",
			wantEnv:    "production",
			wantTxType: "request",
		},
		{
			name: "default transaction type and environment",
			params: TransactionGroupsParams{
				Service: "svc-b",
				Start:   "2024-01-01T00:00:00.000Z",
				End:     "2024-01-02T00:00:00.000Z",
			},
			wantPath:   "/internal/apm/services/svc-b/transactions/groups/main_statistics",
			wantEnv:    "ENVIRONMENT_ALL",
			wantTxType: "request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantPath, r.URL.Path)
				assert.Equal(t, tt.wantEnv, r.URL.Query().Get("environment"))
				assert.Equal(t, tt.wantTxType, r.URL.Query().Get("transactionType"))
				assert.Equal(t, "avg", r.URL.Query().Get("latencyAggregationType"))
				assert.Equal(t, "transactionMetric", r.URL.Query().Get("documentType"))
				assert.Equal(t, "1m", r.URL.Query().Get("rollupInterval"))
				assert.Equal(t, "true", r.URL.Query().Get("useDurationSummary"))

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"transactionGroups": []any{
						map[string]any{
							"name":            "GET /api",
							"latency":         1.5,
							"throughput":      100.0,
							"errorRate":       0.01,
							"impact":          50.0,
							"alertsCount":     0,
							"transactionType": "request",
						},
					},
				})
			})

			groups, err := client.TransactionGroups(context.Background(), tt.params)
			require.NoError(t, err)
			require.Len(t, groups, 1)
			assert.Equal(t, "GET /api", groups[0].Name)
		})
	}
}

func TestTransactionGroups_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.TransactionGroups(context.Background(), TransactionGroupsParams{Service: "svc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction groups")
}

func TestTransactionSamples_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/services/my-svc/transactions/traces/samples", r.URL.Path)
		assert.Equal(t, "production", r.URL.Query().Get("environment"))
		assert.Equal(t, "request", r.URL.Query().Get("transactionType"))
		assert.Equal(t, "GET /api", r.URL.Query().Get("transactionName"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"traceSamples": []any{
				map[string]any{
					"score":         1.0,
					"timestamp":     "2024-01-01T00:00:00.000Z",
					"traceId":       "trace-123",
					"transactionId": "tx-456",
				},
			},
		})
	})

	samples, err := client.TransactionSamples(context.Background(), TransactionSamplesParams{
		Service:         "my-svc",
		Environment:     "production",
		Start:           "2024-01-01T00:00:00.000Z",
		End:             "2024-01-02T00:00:00.000Z",
		TransactionType: "request",
		TransactionName: "GET /api",
	})
	require.NoError(t, err)
	require.Len(t, samples, 1)
	assert.Equal(t, "trace-123", samples[0].TraceID)
	assert.Equal(t, "tx-456", samples[0].TransactionID)
}

func TestTransactionSamples_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.TransactionSamples(context.Background(), TransactionSamplesParams{Service: "svc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction samples")
}
