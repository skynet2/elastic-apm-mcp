package apm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSimpleKuery_Success(t *testing.T) {
	tests := []struct {
		name      string
		kuery     string
		wantField string
		wantValue string
	}{
		{"quoted value", `labels.client_item_id:"111065808"`, "labels.client_item_id", "111065808"},
		{"unquoted value", `service.name:payment-service`, "service.name", "payment-service"},
		{"dotted id field", `trace.id:"abc123"`, "trace.id", "abc123"},
		{"spaces around colon", `service.name : "x"`, "service.name", "x"},
		{"quoted value with space", `labels.note:"a b"`, "labels.note", "a b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, value, ok := parseSimpleKuery(tt.kuery)
			assert.True(t, ok)
			assert.Equal(t, tt.wantField, field)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestParseSimpleKuery_Complex(t *testing.T) {
	tests := []struct {
		name  string
		kuery string
	}{
		{"boolean and", `service.name:"a" and trace.id:"b"`},
		{"boolean or unquoted", `a:b or c:d`},
		{"wildcard", `service.name:pay*`},
		{"parens", `(service.name:"a")`},
		{"range", `transaction.duration.us:>1000`},
		{"no colon", `just-text`},
		{"empty value", `service.name:`},
		{"leading not", `not service.name:"a"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := parseSimpleKuery(tt.kuery)
			assert.False(t, ok)
		})
	}
}

func TestAddKueryFilter_Match_Success(t *testing.T) {
	got := addKueryFilter(nil, `labels.client_item_id:"111065808"`)
	assert.Equal(t, []map[string]any{
		{"match": map[string]any{"labels.client_item_id": "111065808"}},
	}, got)
}

func TestAddKueryFilter_QueryStringFallback_Success(t *testing.T) {
	got := addKueryFilter(nil, `service.name:"a" and trace.id:"b"`)
	assert.Equal(t, []map[string]any{
		{"query_string": map[string]any{"query": `service.name:"a" and trace.id:"b"`, "analyze_wildcard": true}},
	}, got)
}

func TestAddKueryFilter_Empty_Success(t *testing.T) {
	assert.Nil(t, addKueryFilter(nil, "   "))
}
