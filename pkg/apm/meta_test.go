package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceDependencies_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/services/my-svc/dependencies", r.URL.Path)
		assert.Equal(t, "production", r.URL.Query().Get("environment"))
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "20", r.URL.Query().Get("numBuckets"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"serviceDependencies": []any{
				map[string]any{
					"id": "dep-1",
					"location": map[string]any{
						"dependencyName": "postgres",
						"spanType":       "db",
						"spanSubtype":    "postgresql",
						"type":           "external",
						"id":             "dep-loc-1",
					},
				},
			},
		})
	})

	deps, err := client.ServiceDependencies(context.Background(), DependenciesParams{
		Service:     "my-svc",
		Environment: "production",
		Start:       "2024-01-01T00:00:00.000Z",
		End:         "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "dep-1", deps[0].ID)
	assert.Equal(t, "postgres", deps[0].Location.DependencyName)
}

func TestServiceDependencies_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.ServiceDependencies(context.Background(), DependenciesParams{Service: "svc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service dependencies")
}

func TestEnvironments_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/environments", r.URL.Path)
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "", r.URL.Query().Get("environment"), "must not send environment param")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"environments": []any{"production", "staging"},
		})
	})

	envs, err := client.Environments(context.Background(), "2024-01-01T00:00:00.000Z", "2024-01-02T00:00:00.000Z")
	require.NoError(t, err)
	require.Len(t, envs, 2)
	assert.Equal(t, "production", envs[0])
}

func TestEnvironments_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.Environments(context.Background(), "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "environments")
}

func TestGetAPMIndices_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm-sources/settings/apm-indices", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(APMIndices{
			Transaction: "traces-apm*",
			Span:        "traces-apm*",
			Error:       "logs-apm.error*",
			Metric:      "metrics-apm*",
			Onboarding:  "logs-apm.onboarding*",
			Sourcemap:   "logs-apm.sourcemap*",
		})
	})

	indices, err := client.GetAPMIndices(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "traces-apm*", indices.Transaction)
	assert.Equal(t, "logs-apm.error*", indices.Error)
}

func TestGetAPMIndices_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.GetAPMIndices(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apm indices")
}
