package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skynet2/elastic-apm-mcp/cmd/elastic-apm-mcp/mcp/mocks"
	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

func TestListIndicesHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		pattern  string
		response []string
		wantJSON string
	}{
		{
			name:     "explicit_pattern",
			args:     map[string]any{"pattern": "app-logs-*"},
			pattern:  "app-logs-*",
			response: []string{"app-logs-000001"},
			wantJSON: `"app-logs-000001"`,
		},
		{
			name:     "default_pattern",
			args:     map[string]any{},
			pattern:  "",
			response: []string{"index-a"},
			wantJSON: `"index-a"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().ListIndices(gomock.Any(), tc.pattern).Return(tc.response, nil)

			h := listIndicesHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestListIndicesHandler_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockAPMClient(ctrl)
	m.EXPECT().ListIndices(gomock.Any(), "bad-*").Return(nil, errors.New("list fail"))

	h := listIndicesHandler(m, zerolog.Nop())
	result, err := h(context.Background(), newReq(map[string]any{"pattern": "bad-*"}))

	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, textContent(result), "list fail")
}

func TestDescribeFieldsHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	response := []apm.Field{
		{Name: "service_name", Type: "string", ESTypes: []string{"text"}, Searchable: true, Aggregatable: false},
		{Name: "trace_id", Type: "string", ESTypes: []string{"keyword"}, Searchable: true, Aggregatable: true},
	}

	m := mocks.NewMockAPMClient(ctrl)
	m.EXPECT().DescribeFields(gomock.Any(), "app-logs-*").Return(response, nil)

	h := describeFieldsHandler(m, zerolog.Nop())
	result, err := h(context.Background(), newReq(map[string]any{"pattern": "app-logs-*"}))

	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, textContent(result), `"service_name"`)
	assert.Contains(t, textContent(result), `"keyword"`)
}

func TestDescribeFieldsHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name:        "missing_pattern",
			args:        map[string]any{},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"pattern"`,
		},
		{
			name: "client_error",
			args: map[string]any{"pattern": "app-logs-*"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().DescribeFields(gomock.Any(), "app-logs-*").Return(nil, errors.New("fields fail"))
			},
			errContains: "fields fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := describeFieldsHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
