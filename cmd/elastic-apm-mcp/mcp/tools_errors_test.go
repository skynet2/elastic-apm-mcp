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

func TestErrorGroupsHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.ErrorGroupsParams
		response map[string]any
		wantJSON string
	}{
		{
			name: "basic",
			args: map[string]any{"service": "svc"},
			params: apm.ErrorGroupsParams{
				Service: "svc",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: map[string]any{"errorGroups": []any{}},
			wantJSON: `"errorGroups"`,
		},
		{
			name: "with_kuery",
			args: map[string]any{"service": "svc", "kuery": `transaction.name:"POST /pay"`},
			params: apm.ErrorGroupsParams{
				Service: "svc",
				Kuery:   `transaction.name:"POST /pay"`,
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: map[string]any{"errorGroups": []any{}},
			wantJSON: `"errorGroups"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().ErrorGroups(gomock.Any(), tc.params).Return(tc.response, nil)

			h := errorGroupsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestErrorGroupsHandler_Failure(t *testing.T) {
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
				m.EXPECT().ErrorGroups(gomock.Any(), apm.ErrorGroupsParams{
					Service: "svc",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("err groups fail"))
			},
			errContains: "err groups fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := errorGroupsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestErrorGetHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.ErrorGetParams
		response []map[string]any
		wantJSON string
	}{
		{
			name:   "by_error_id",
			args:   map[string]any{"error_id": "err-123"},
			params: apm.ErrorGetParams{ErrorID: "err-123"},
			response: []map[string]any{
				{"error": map[string]any{"id": "err-123"}},
			},
			wantJSON: `"error"`,
		},
		{
			name:   "by_grouping_key",
			args:   map[string]any{"grouping_key": "gk-abc"},
			params: apm.ErrorGetParams{GroupingKey: "gk-abc"},
			response: []map[string]any{
				{"error": map[string]any{"grouping_key": "gk-abc"}},
			},
			wantJSON: `"gk-abc"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().ErrorGet(gomock.Any(), tc.params).Return(tc.response, nil)

			h := errorGetHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestErrorGetHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name: "client_returns_error",
			args: map[string]any{},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().ErrorGet(gomock.Any(), apm.ErrorGetParams{}).Return(nil, errors.New("errorId or groupingKey required"))
			},
			errContains: "required",
		},
		{
			name: "client_error_with_id",
			args: map[string]any{"error_id": "bad"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().ErrorGet(gomock.Any(), apm.ErrorGetParams{ErrorID: "bad"}).Return(nil, errors.New("not found"))
			},
			errContains: "not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := errorGetHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
