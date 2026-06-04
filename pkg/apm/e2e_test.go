//go:build e2e

package apm

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_E2E_Parity(t *testing.T) {
	apmURL := os.Getenv("APM_URL")
	apiKey := os.Getenv("APM_API_KEY")
	if apmURL == "" || apiKey == "" {
		t.Skip("set APM_URL and APM_API_KEY to run e2e parity")
	}

	headers := map[string]string{}
	if raw := os.Getenv("APM_HEADERS"); raw != "" {
		for _, pair := range strings.Split(raw, ",") {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				headers[kv[0]] = kv[1]
			}
		}
	}

	client := New(Config{
		BaseURL: apmURL,
		APIKey:  apiKey,
		Headers: headers,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	})

	ctx := context.Background()
	now := time.Now()

	t.Run("indices", func(t *testing.T) {
		indices, err := client.GetAPMIndices(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, indices.Transaction)
		assert.NotEmpty(t, indices.Error)
	})

	t.Run("environments", func(t *testing.T) {
		start, err := ResolveTime("now-24h", now)
		require.NoError(t, err)
		end, err := ResolveTime("now", now)
		require.NoError(t, err)

		_, err = client.Environments(ctx, FormatISO(start), FormatISO(end))
		require.NoError(t, err)
	})

	t.Run("services", func(t *testing.T) {
		start, err := ResolveTime("now-15m", now)
		require.NoError(t, err)
		end, err := ResolveTime("now", now)
		require.NoError(t, err)

		svcs, err := client.ServiceList(ctx, ServiceListParams{
			Start: FormatISO(start),
			End:   FormatISO(end),
		})
		require.NoError(t, err)
		if len(svcs) > 0 {
			assert.NotEmpty(t, svcs[0].ServiceName)
		}
	})
}
