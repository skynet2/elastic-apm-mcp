package apm

import "encoding/json"

type Service struct {
	ServiceName          string   `json:"serviceName"`
	AgentName            string   `json:"agentName"`
	Environments         []string `json:"environments"`
	Latency              float64  `json:"latency"`
	Throughput           float64  `json:"throughput"`
	TransactionErrorRate float64  `json:"transactionErrorRate"`
	TransactionType      string   `json:"transactionType"`
}

type TransactionGroup struct {
	Name            string  `json:"name"`
	Latency         float64 `json:"latency"`
	Throughput      float64 `json:"throughput"`
	ErrorRate       float64 `json:"errorRate"`
	Impact          float64 `json:"impact"`
	AlertsCount     int     `json:"alertsCount"`
	TransactionType string  `json:"transactionType"`
}

type TraceSample struct {
	Score         float64 `json:"score"`
	Timestamp     string  `json:"timestamp"`
	TraceID       string  `json:"traceId"`
	TransactionID string  `json:"transactionId"`
}

type TraceItemErrorRef struct {
	ErrorDocID    string `json:"errorDocId"`
	ErrorDocIndex string `json:"errorDocIndex"`
}

type TraceItem struct {
	ID                 string              `json:"id"`
	ParentID           string              `json:"parentId"`
	Name               string              `json:"name"`
	ServiceName        string              `json:"serviceName"`
	ServiceEnvironment string              `json:"serviceEnvironment"`
	AgentName          string              `json:"agentName"`
	DocType            string              `json:"docType"`
	Duration           int64               `json:"duration"`
	Result             string              `json:"result"`
	TimestampUs        int64               `json:"timestampUs"`
	TraceID            string              `json:"traceId"`
	Errors             []TraceItemErrorRef `json:"errors"`
}

type TraceError struct {
	ID    string `json:"id"`
	Index string `json:"index"`
	Error struct {
		Culprit     string `json:"culprit"`
		GroupingKey string `json:"grouping_key"`
		ID          string `json:"id"`
		Exception   struct {
			Handled bool   `json:"handled"`
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"exception"`
	} `json:"error"`
	Service struct {
		Name string `json:"name"`
	} `json:"service"`
	Transaction struct {
		ID string `json:"id"`
	} `json:"transaction"`
	Span struct {
		ID string `json:"id"`
	} `json:"span"`
	Trace struct {
		ID string `json:"id"`
	} `json:"trace"`
	Parent struct {
		ID string `json:"id"`
	} `json:"parent"`
	Timestamp struct {
		Us int64 `json:"us"`
	} `json:"timestamp"`
}

type Trace struct {
	TraceItems       []TraceItem    `json:"traceItems"`
	Errors           []TraceError   `json:"errors"`
	EntryTransaction map[string]any `json:"entryTransaction"`
	TraceDocsTotal   int            `json:"traceDocsTotal"`
	MaxTraceItems    int            `json:"maxTraceItems"`
}

type APMIndices struct {
	Transaction string `json:"transaction"`
	Span        string `json:"span"`
	Error       string `json:"error"`
	Metric      string `json:"metric"`
	Onboarding  string `json:"onboarding"`
	Sourcemap   string `json:"sourcemap"`
}

type ServiceDependency struct {
	ID       string `json:"id"`
	Location struct {
		DependencyName string `json:"dependencyName"`
		SpanType       string `json:"spanType"`
		SpanSubtype    string `json:"spanSubtype"`
		Type           string `json:"type"`
		ID             string `json:"id"`
	} `json:"location"`
	CurrentStats  json.RawMessage `json:"currentStats"`
	PreviousStats json.RawMessage `json:"previousStats"`
}
