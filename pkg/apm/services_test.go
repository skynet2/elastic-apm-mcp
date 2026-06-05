package apm

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceList_Success(t *testing.T) {
	tests := []struct {
		name      string
		params    ServiceListParams
		wantEnv   string
		wantKuery string
		wantStart string
		wantEnd   string
	}{
		{
			name: "explicit environment",
			params: ServiceListParams{
				Environment: "production",
				Kuery:       "service.name:foo",
				Start:       "2024-01-01T00:00:00.000Z",
				End:         "2024-01-02T00:00:00.000Z",
			},
			wantEnv:   "production",
			wantKuery: "service.name:foo",
			wantStart: "2024-01-01T00:00:00.000Z",
			wantEnd:   "2024-01-02T00:00:00.000Z",
		},
		{
			name: "empty environment defaults",
			params: ServiceListParams{
				Start: "2024-01-01T00:00:00.000Z",
				End:   "2024-01-02T00:00:00.000Z",
			},
			wantEnv:   "ENVIRONMENT_ALL",
			wantKuery: "",
			wantStart: "2024-01-01T00:00:00.000Z",
			wantEnd:   "2024-01-02T00:00:00.000Z",
		},
	}

	data, err := os.ReadFile("testdata/services.json")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/internal/apm/services", r.URL.Path)
				assert.Equal(t, tt.wantEnv, r.URL.Query().Get("environment"))
				assert.Equal(t, tt.wantKuery, r.URL.Query().Get("kuery"))
				assert.Equal(t, tt.wantStart, r.URL.Query().Get("start"))
				assert.Equal(t, tt.wantEnd, r.URL.Query().Get("end"))
				assert.Equal(t, "serviceTransactionMetric", r.URL.Query().Get("documentType"))
				assert.Equal(t, "1m", r.URL.Query().Get("rollupInterval"))
				assert.Equal(t, "true", r.URL.Query().Get("useDurationSummary"))
				assert.Equal(t, "1", r.URL.Query().Get("probability"))

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
			})

			svcs, err := client.ServiceList(context.Background(), tt.params)
			require.NoError(t, err)
			require.Len(t, svcs, 1)
			assert.Equal(t, "payment-service", svcs[0].ServiceName)
			assert.Equal(t, "go", svcs[0].AgentName)
		})
	}
}

func TestServiceList_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.ServiceList(context.Background(), ServiceListParams{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service list")
}
