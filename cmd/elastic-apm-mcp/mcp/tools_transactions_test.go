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

func TestTransactionGroupsHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.TransactionGroupsParams
		response []apm.TransactionGroup
		wantJSON string
	}{
		{
			name: "basic",
			args: map[string]any{"service": "svc"},
			params: apm.TransactionGroupsParams{
				Service: "svc",
				Start:   defaultStart,
				End:     defaultEnd,
			},
			response: []apm.TransactionGroup{{Name: "GET /api"}},
			wantJSON: `"GET /api"`,
		},
		{
			name: "with_tx_type",
			args: map[string]any{"service": "svc", "transaction_type": "worker"},
			params: apm.TransactionGroupsParams{
				Service:         "svc",
				TransactionType: "worker",
				Start:           defaultStart,
				End:             defaultEnd,
			},
			response: []apm.TransactionGroup{{Name: "process_job"}},
			wantJSON: `"process_job"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().TransactionGroups(gomock.Any(), tc.params).Return(tc.response, nil)

			h := transactionGroupsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestTransactionGroupsHandler_Failure(t *testing.T) {
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
				m.EXPECT().TransactionGroups(gomock.Any(), apm.TransactionGroupsParams{
					Service: "svc",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("tx groups fail"))
			},
			errContains: "tx groups fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := transactionGroupsHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}

func TestTransactionSamplesHandler_Success(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]any
		params   apm.TransactionSamplesParams
		response []apm.TraceSample
		wantJSON string
	}{
		{
			name: "basic",
			args: map[string]any{"service": "svc", "transaction_name": "GET /api"},
			params: apm.TransactionSamplesParams{
				Service:         "svc",
				TransactionName: "GET /api",
				Start:           defaultStart,
				End:             defaultEnd,
			},
			response: []apm.TraceSample{{TraceID: "trace-abc"}},
			wantJSON: `"trace-abc"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			m.EXPECT().TransactionSamples(gomock.Any(), tc.params).Return(tc.response, nil)

			h := transactionSamplesHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Contains(t, textContent(result), tc.wantJSON)
		})
	}
}

func TestTransactionSamplesHandler_Failure(t *testing.T) {
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
				m.EXPECT().TransactionSamples(gomock.Any(), apm.TransactionSamplesParams{
					Service: "svc",
					Start:   defaultStart,
					End:     defaultEnd,
				}).Return(nil, errors.New("samples fail"))
			},
			errContains: "samples fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := transactionSamplesHandler(m, zerolog.Nop(), fixedClock)
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
