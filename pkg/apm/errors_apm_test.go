package apm

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorGroups_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/apm/services/my-svc/errors/groups/main_statistics", r.URL.Path)
		assert.Equal(t, "production", r.URL.Query().Get("environment"))
		assert.Equal(t, "2024-01-01T00:00:00.000Z", r.URL.Query().Get("start"))
		assert.Equal(t, "2024-01-02T00:00:00.000Z", r.URL.Query().Get("end"))
		assert.Equal(t, "", r.URL.Query().Get("maxNumberOfErrorGroups"), "must not send maxNumberOfErrorGroups")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorGroups": []any{
				map[string]any{"groupId": "g1", "name": "NullPointerException"},
			},
		})
	})

	result, err := client.ErrorGroups(context.Background(), ErrorGroupsParams{
		Service:     "my-svc",
		Environment: "production",
		Start:       "2024-01-01T00:00:00.000Z",
		End:         "2024-01-02T00:00:00.000Z",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
	groups, ok := result["errorGroups"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 1)
}

func TestErrorGroups_Failure(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := client.ErrorGroups(context.Background(), ErrorGroupsParams{Service: "svc"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error groups")
}
