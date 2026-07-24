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

func TestESQLHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	query := "FROM traces-apm* | STATS c = COUNT(*) BY service.name"
	response := apm.ESQLResult{
		Columns:  []apm.ESQLColumn{{Name: "c", Type: "long"}, {Name: "service.name", Type: "keyword"}},
		Rows:     []map[string]any{{"c": float64(5), "service.name": "svc-a"}},
		RowCount: 1,
	}

	m := mocks.NewMockAPMClient(ctrl)
	m.EXPECT().ESQL(gomock.Any(), query).Return(response, nil)

	h := esqlHandler(m, zerolog.Nop())
	result, err := h(context.Background(), newReq(map[string]any{"query": query}))

	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, textContent(result), `"row_count"`)
	assert.Contains(t, textContent(result), `"svc-a"`)
}

func TestESQLHandler_Failure(t *testing.T) {
	cases := []struct {
		name        string
		args        map[string]any
		setupMock   func(*mocks.MockAPMClient)
		errContains string
	}{
		{
			name:        "missing_query",
			args:        map[string]any{},
			setupMock:   func(m *mocks.MockAPMClient) {},
			errContains: `"query"`,
		},
		{
			name: "client_error",
			args: map[string]any{"query": "FROM traces-apm* | LIMIT 1"},
			setupMock: func(m *mocks.MockAPMClient) {
				m.EXPECT().ESQL(gomock.Any(), "FROM traces-apm* | LIMIT 1").
					Return(apm.ESQLResult{}, errors.New("esql fail"))
			},
			errContains: "esql fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mocks.NewMockAPMClient(ctrl)
			tc.setupMock(m)

			h := esqlHandler(m, zerolog.Nop())
			result, err := h(context.Background(), newReq(tc.args))

			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, textContent(result), tc.errContains)
		})
	}
}
