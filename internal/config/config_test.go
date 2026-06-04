package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skynet2/elastic-apm-mcp/internal/config"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/config.yaml"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoad_Success(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		envs      map[string]string
		wantURL   string
		wantKey   string
		wantTO    time.Duration
		wantLevel string
		wantHdrs  map[string]string
	}{
		{
			name: "yaml only",
			yaml: `
url: https://kibana.example.com
api_key: my-api-key
timeout: 45s
log_level: debug
`,
			wantURL:   "https://kibana.example.com",
			wantKey:   "my-api-key",
			wantTO:    45 * time.Second,
			wantLevel: "debug",
		},
		{
			name: "env overrides url",
			yaml: `
url: https://kibana.example.com
api_key: my-api-key
`,
			envs:      map[string]string{"APM_URL": "https://override.example.com"},
			wantURL:   "https://override.example.com",
			wantKey:   "my-api-key",
			wantTO:    30 * time.Second,
			wantLevel: "info",
		},
		{
			name:      "env only",
			yaml:      "",
			envs:      map[string]string{"APM_URL": "https://env.example.com", "APM_API_KEY": "env-key"},
			wantURL:   "https://env.example.com",
			wantKey:   "env-key",
			wantTO:    30 * time.Second,
			wantLevel: "info",
		},
		{
			name: "defaults timeout and log_level",
			yaml: `
url: https://kibana.example.com
api_key: my-api-key
`,
			wantURL:   "https://kibana.example.com",
			wantKey:   "my-api-key",
			wantTO:    30 * time.Second,
			wantLevel: "info",
		},
		{
			name: "yaml headers map",
			yaml: `
url: https://kibana.example.com
api_key: my-api-key
headers:
  CF-Access-Client-Id: cid-value
  CF-Access-Client-Secret: csecret-value
`,
			wantURL:   "https://kibana.example.com",
			wantKey:   "my-api-key",
			wantTO:    30 * time.Second,
			wantLevel: "info",
			wantHdrs: map[string]string{
				"cf-access-client-id":     "cid-value",
				"cf-access-client-secret": "csecret-value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			path := ""
			if tc.yaml != "" {
				path = writeYAML(t, tc.yaml)
			}

			cfg, err := config.Load(path)
			require.NoError(t, err)
			assert.Equal(t, tc.wantURL, cfg.URL)
			assert.Equal(t, tc.wantKey, cfg.APIKey)
			assert.Equal(t, tc.wantTO, cfg.Timeout)
			assert.Equal(t, tc.wantLevel, cfg.LogLevel)
			if tc.wantHdrs != nil {
				assert.Equal(t, tc.wantHdrs, cfg.Headers)
			}
		})
	}
}

func TestLoad_Failure(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		envs    map[string]string
		wantErr string
	}{
		{
			name:    "missing url",
			yaml:    "api_key: my-api-key",
			wantErr: "config: url is required",
		},
		{
			name:    "missing api_key",
			yaml:    "url: https://kibana.example.com",
			wantErr: "config: api_key is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			path := writeYAML(t, tc.yaml)
			_, err := config.Load(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
