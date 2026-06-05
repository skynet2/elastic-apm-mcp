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

func TestTraceGetHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		traceID  string
		params   apm.TraceParams
		response apm.Trace
		wantJSON string
	}{
		{
			name:    "basic",
			args:    map[string]any{"trace_id": "trace-123"},
			traceID: "trace-123",
			params: apm.TraceParams{
				Start: defaultStart,
				End:   defaultEnd,
			},
			response: apm.Trace{TraceDocsTotal: 5},
			wantJSON: `"traceDocsTotal"`,
		},
		{
			name: "with_entry_transaction",
			args: map[string]any{
				"trace_id":             "trace-456",
				"entry_transaction_id": "tx-789",
			},
			traceID: "trace-456",
			params: apm.TraceParams{
				Start:              defaultStart,
				End:                defaultEnd,
				EntryTransactionID: "tx-789",
			},
			response: apm.Trace{TraceDocsTotal: 3},
			wantJSON: `"traceDocsTotal"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().TraceGet(gomock.Any(), tc.traceID, tc.params).Return(tc.response, nil)

			h := traceGetHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestTraceGetHandler_Failure(t *testing.T) {
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
			args: map[string]any{"trace_id": "trace-123"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().TraceGet(gomock.Any(), "trace-123", apm.TraceParams{
					Start: defaultStart,
					End:   defaultEnd,
				}).Return(apm.Trace{}, errors.New("trace fail"))
			},
			errContains: "trace fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := traceGetHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestTraceSearchHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.TraceSearchParams
		response []map[string]any
		wantJSON string
	}{
		{
			name: "with_kuery",
			args: map[string]any{"kuery": `labels.env:"prod"`, "service": "svc"},
			params: apm.TraceSearchParams{
				Kuery:   `labels.env:"prod"`,
				Service: "svc",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{{"trace": map[string]any{"id": "abc"}}},
			wantJSON: `"trace"`,
		},
		{
			name: "default_size",
			args: map[string]any{"service": "svc"},
			params: apm.TraceSearchParams{
				Service: "svc",
				Size:    0,
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{{"trace": map[string]any{"id": "def"}}},
			wantJSON: `"trace"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().TraceSearch(gomock.Any(), tc.params).Return(tc.response, nil)

			h := traceSearchHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestTraceSearchHandler_Failure(t *testing.T) {
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
				m.EXPECT().TraceSearch(gomock.Any(), apm.TraceSearchParams{
					Start: defaultStart,
					End:   defaultEnd,
				}).Return(nil, errors.New("search fail"))
			},
			errContains: "search fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := traceSearchHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
