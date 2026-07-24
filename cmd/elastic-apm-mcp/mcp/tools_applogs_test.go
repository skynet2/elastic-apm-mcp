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

func TestAppLogsSearchHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.AppLogsParams
		response []map[string]any
		wantJSON string
	}{
		{
			name: "by_service_and_message",
			args: map[string]any{"service": "payment-service", "message": "processing request"},
			params: apm.AppLogsParams{
				Service: "payment-service",
				Message: "processing request",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{{"response_body": `{"status":"ok"}`}},
			wantJSON: `"response_body"`,
		},
		{
			name: "by_trace_and_kuery",
			args: map[string]any{"trace_id": "trace-abc", "kuery": "vendor:acme", "size": float64(20)},
			params: apm.AppLogsParams{
				TraceID: "trace-abc",
				Kuery:   "vendor:acme",
				Size:    20,
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{{"vendor": "acme"}},
			wantJSON: `"acme"`,
		},
		{
			name: "by_pod_namespace_level",
			args: map[string]any{"pod": "payment-service-1", "namespace": "staging", "level": "info", "container": "payment-service"},
			params: apm.AppLogsParams{
				Pod:       "payment-service-1",
				Namespace: "staging",
				Level:     "info",
				Container: "payment-service",
				Start:     defaultStart,
				End:       defaultEnd,
			},
			response: []map[string]any{},
			wantJSON: `[]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().AppLogsSearch(gomock.Any(), tc.params).Return(tc.response, nil)

			h := appLogsSearchHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestAppLogsSearchHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name: "client_error",
			args: map[string]any{"service": "payment-service"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().AppLogsSearch(gomock.Any(), apm.AppLogsParams{
					Service: "payment-service",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("applogs fail"))
			},
			errContains: "applogs fail",
		},
		{
			name:        "invalid_start",
			args:        map[string]any{"start": "not-a-time"},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: "invalid start",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := appLogsSearchHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestTraceLogsHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.TraceLogsParams
		response []map[string]any
		wantJSON string
	}{
		{
			name: "by_trace_id",
			args: map[string]any{"trace_id": "trace-abc"},
			params: apm.TraceLogsParams{
				TraceID: "trace-abc",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{{"message": "item not found", "_index": "logs-apm.error"}},
			wantJSON: `"_index"`,
		},
		{
			name: "with_size",
			args: map[string]any{"trace_id": "trace-xyz", "size": float64(25)},
			params: apm.TraceLogsParams{
				TraceID: "trace-xyz",
				Size:    25,
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{},
			wantJSON: `[]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().TraceLogs(gomock.Any(), tc.params).Return(tc.response, nil)

			h := traceLogsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestTraceLogsHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name:        "missing_trace_id",
			args:        map[string]any{},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"trace_id"`,
		},
		{
			name: "client_error",
			args: map[string]any{"trace_id": "trace-abc"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().TraceLogs(gomock.Any(), apm.TraceLogsParams{
					TraceID: "trace-abc",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("tracelogs fail"))
			},
			errContains: "tracelogs fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := traceLogsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
