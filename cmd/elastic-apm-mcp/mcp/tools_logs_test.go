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

func TestLogsSearchHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.LogsParams
		response []map[string]any
		wantJSON string
	}{
		{
			name: "by_trace_id",
			args: map[string]any{"trace_id": "trace-abc"},
			params: apm.LogsParams{
				TraceID: "trace-abc",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []map[string]any{{"message": "error occurred"}},
			wantJSON: `"message"`,
		},
		{
			name: "by_transaction_and_span",
			args: map[string]any{"transaction_id": "tx-1", "span_id": "sp-2"},
			params: apm.LogsParams{
				TransactionID: "tx-1",
				SpanID:        "sp-2",
				Start:         defaultStart,
				End:           defaultEnd,
			},
			response: []map[string]any{{"log": map[string]any{"level": "error"}}},
			wantJSON: `"log"`,
		},
		{
			name: "with_size",
			args: map[string]any{"trace_id": "trace-xyz", "size": float64(10)},
			params: apm.LogsParams{
				TraceID: "trace-xyz",
				Size:    10,
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
			m.EXPECT().LogsSearch(gomock.Any(), tc.params).Return(tc.response, nil)

			h := logsSearchHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestLogsSearchHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name: "client_error",
			args: map[string]any{"trace_id": "trace-abc"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().LogsSearch(gomock.Any(), apm.LogsParams{
					TraceID: "trace-abc",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("logs fail"))
			},
			errContains: "logs fail",
		},
		{
			name: "invalid_end",
			args: map[string]any{"end": "not-a-time"},
			setupMock: func(m *mocks.MockAPMClient) {
			},
			errContains: "invalid end",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := logsSearchHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestESSearchHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		index    string
		body     map[string]any
		response []map[string]any
		wantJSON string
	}{
		{
			name:  "basic_query",
			index: "traces-apm*",
			args: map[string]any{
				"index": "traces-apm*",
				"query": map[string]any{
					"query": map[string]any{"match_all": map[string]any{}},
				},
			},
			body: map[string]any{
				"query": map[string]any{"match_all": map[string]any{}},
			},
			response: []map[string]any{{"_id": "doc1"}},
			wantJSON: `"_id"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().RawSearch(gomock.Any(), tc.index, tc.body).Return(tc.response, nil)

			h := esSearchHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestESSearchHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name:        "missing_index",
			args:        map[string]any{"query": map[string]any{}},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"index"`,
		},
		{
			name:        "missing_query",
			args:        map[string]any{"index": "traces-apm*"},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"query"`,
		},
		{
			name: "client_error",
			args: map[string]any{
				"index": "traces-apm*",
				"query": map[string]any{"query": map[string]any{}},
			},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().RawSearch(gomock.Any(), "traces-apm*", map[string]any{"query": map[string]any{}}).
					Return(nil, errors.New("es fail"))
			},
			errContains: "es fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := esSearchHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
