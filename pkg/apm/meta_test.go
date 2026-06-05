package apm

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceDependencies_Success(t *testing.T) {
	data, err := os.ReadFile("testdata/dependencies.json")
	require.NoError(t, err)

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/services/payment-service/dependencies", r.URL.Path)
		assert.Equal(t, "production", r.URL.Query().Get("environment"))
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "20", r.URL.Query().Get("numBuckets"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	deps, err := client.ServiceDependencies(context.Background(), DependenciesParams{
		Service:     "payment-service",
		Environment: "production",
		Start:       "2024-01-01T00:00:00.000Z",
		End:         "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "dep0000000000001", deps[0].ID)
	assert.Equal(t, "api.example.com:443", deps[0].Location.DependencyName)
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
	data, err := os.ReadFile("testdata/environments.json")
	require.NoError(t, err)

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/environments", r.URL.Path)
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "", r.URL.Query().Get("environment"), "must not send environment param")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	envs, err := client.Environments(context.Background(), "2024-01-01T00:00:00.000Z", "2024-01-02T00:00:00.000Z")
	require.NoError(t, err)
	require.Len(t, envs, 2)
	assert.Equal(t, "prod", envs[0])
	assert.Equal(t, "staging", envs[1])
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
	data, err := os.ReadFile("testdata/apm_indices.json")
	require.NoError(t, err)

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm-sources/settings/apm-indices", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	indices, err := client.GetAPMIndices(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "traces-apm*,apm-*,traces-*.otel-*", indices.Transaction)
	assert.Equal(t, "logs-apm*,apm-*,logs-*.otel-*", indices.Error)
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
