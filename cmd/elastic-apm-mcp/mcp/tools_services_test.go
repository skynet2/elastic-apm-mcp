package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skynet2/elastic-apm-mcp/cmd/elastic-apm-mcp/mcp/mocks"
	"github.com/skynet2/elastic-apm-mcp/pkg/apm"
)

var fixedNow = time.Date(2026, 6, 4, 16, 49, 0, 0, time.UTC)

func fixedClock() time.Time { return fixedNow }

const (
	defaultStart = "2026-06-04T16:34:00Z"
	defaultEnd   = "2026-06-04T16:49:00Z"
)

func newReq(args map[string]any) mcplib.CallToolRequest {
	return mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Arguments: args,
		},
	}
}

func textContent(result *mcplib.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	tc, ok := result.Content[0].(mcplib.TextContent)
	if !ok {
		return ""
	}
	return tc.Text
}

func TestServiceListHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.ServiceListParams
		response []apm.Service
		wantJSON string
	}{
		{
			name: "default_range",
			args: map[string]any{},
			params: apm.ServiceListParams{
				Start: defaultStart,
				End:   defaultEnd,
			},
			response: []apm.Service{{ServiceName: "my-svc"}},
			wantJSON: `"my-svc"`,
		},
		{
			name: "with_env_and_kuery",
			args: map[string]any{"environment": "prod", "kuery": "service.name:foo"},
			params: apm.ServiceListParams{
				Environment: "prod",
				Kuery:       "service.name:foo",
				Start:       defaultStart,
				End:         defaultEnd,
			},
			response: []apm.Service{{ServiceName: "foo"}},
			wantJSON: `"foo"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().ServiceList(gomock.Any(), tc.params).Return(tc.response, nil)

			h := serviceListHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestServiceListHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name: "client_error",
			args: map[string]any{},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().ServiceList(gomock.Any(), apm.ServiceListParams{
					Start: defaultStart,
					End:   defaultEnd,
				}).Return(nil, errors.New("upstream down"))
			},
			errContains: "upstream down",
		},
		{
			name: "invalid_start",
			args: map[string]any{"start": "bad-time"},
			setupMock: func(m *mocks.MockAPMClient) {
			},
			errContains: "invalid start",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := serviceListHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestServiceMetricsHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.ServiceMetricsParams
		response map[string]any
		wantJSON string
	}{
		{
			name: "latency_metric",
			args: map[string]any{"service": "svc", "metric": "latency"},
			params: apm.ServiceMetricsParams{
				Service: "svc",
				Metric:  "latency",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: map[string]any{"currentPeriod": map[string]any{}},
			wantJSON: `"currentPeriod"`,
		},
		{
			name: "throughput_with_offset",
			args: map[string]any{"service": "svc", "metric": "throughput", "offset": "1d"},
			params: apm.ServiceMetricsParams{
				Service: "svc",
				Metric:  "throughput",
				Offset:  "1d",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: map[string]any{"throughput": []any{}},
			wantJSON: `"throughput"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().ServiceMetrics(gomock.Any(), tc.params).Return(tc.response, nil)

			h := serviceMetricsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestServiceMetricsHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name:        "missing_service",
			args:        map[string]any{"metric": "latency"},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"service"`,
		},
		{
			name:        "missing_metric",
			args:        map[string]any{"service": "svc"},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"metric"`,
		},
		{
			name: "client_error",
			args: map[string]any{"service": "svc", "metric": "latency"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().ServiceMetrics(gomock.Any(), apm.ServiceMetricsParams{
					Service: "svc",
					Metric:  "latency",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("apm error"))
			},
			errContains: "apm error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := serviceMetricsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestServiceDependenciesHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.DependenciesParams
		response []apm.ServiceDependency
		wantJSON string
	}{
		{
			name: "basic",
			args: map[string]any{"service": "svc"},
			params: apm.DependenciesParams{
				Service: "svc",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []apm.ServiceDependency{{ID: "dep1"}},
			wantJSON: `"dep1"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().ServiceDependencies(gomock.Any(), tc.params).Return(tc.response, nil)

			h := serviceDependenciesHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestServiceDependenciesHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name:        "missing_service",
			args:        map[string]any{},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"service"`,
		},
		{
			name: "client_error",
			args: map[string]any{"service": "svc"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().ServiceDependencies(gomock.Any(), apm.DependenciesParams{
					Service: "svc",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("fail"))
			},
			errContains: "fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := serviceDependenciesHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestEnvironmentsHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		response []string
		wantJSON string
	}{
		{
			name:     "default_range",
			args:     map[string]any{},
			response: []string{"prod", "staging"},
			wantJSON: `"prod"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().Environments(gomock.Any(), defaultStart, defaultEnd).Return(tc.response, nil)

			h := environmentsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestEnvironmentsHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name: "client_error",
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().Environments(gomock.Any(), defaultStart, defaultEnd).Return(nil, errors.New("env fail"))
			},
			errContains: "env fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := environmentsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(map[string]any{}))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestAPMIndicesHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		response apm.APMIndices
		wantJSON string
	}{
		{
			name:     "basic",
			response: apm.APMIndices{Transaction: "traces-apm*"},
			wantJSON: `"traces-apm*"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().GetAPMIndices(gomock.Any()).Return(tc.response, nil)

			h := apmIndicesHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(map[string]any{}))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestAPMIndicesHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name: "client_error",
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().GetAPMIndices(gomock.Any()).Return(apm.APMIndices{}, errors.New("indices fail"))
			},
			errContains: "indices fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := apmIndicesHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(map[string]any{}))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
