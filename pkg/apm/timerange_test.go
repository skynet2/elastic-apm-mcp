package apm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ref = time.Date(2026, 6, 4, 16, 49, 0, 0, time.UTC)

func TestResolveTime_Success(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{name: "now", input: "now", want: ref},
		{name: "now-15m", input: "now-15m", want: ref.Add(-15 * time.Minute)},
		{name: "now-1h", input: "now-1h", want: ref.Add(-1 * time.Hour)},
		{name: "now-1d", input: "now-1d", want: ref.Add(-24 * time.Hour)},
		{name: "now-30s", input: "now-30s", want: ref.Add(-30 * time.Second)},
		{name: "rfc3339 literal", input: "2026-06-04T16:00:00Z", want: time.Date(2026, 6, 4, 16, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveTime(tc.input, ref)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveTime_Failure(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "garbage string", input: "garbage"},
		{name: "unknown unit", input: "now-5x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveTime(tc.input, ref)
			require.Error(t, err)
		})
	}
}

func TestISO(t *testing.T) {
	got := ISO(time.Date(2026, 6, 4, 16, 49, 0, 0, time.UTC))
	assert.Equal(t, "2026-06-04T16:49:00Z", got)
}
