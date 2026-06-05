package apm

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ResolveTime(s string, now time.Time) (time.Time, error) {
	if s == "now" {
		return now, nil
	}

	if strings.HasPrefix(s, "now-") {
		return parseRelative(s[4:], now)
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("apm: parse time %q: %w", s, err)
	}
	return t, nil
}

func parseRelative(expr string, now time.Time) (time.Time, error) {
	if len(expr) < 2 {
		return time.Time{}, fmt.Errorf("apm: invalid relative time expression: %q", expr)
	}

	unit := string(expr[len(expr)-1])
	n, err := strconv.Atoi(expr[:len(expr)-1])
	if err != nil {
		return time.Time{}, fmt.Errorf("apm: invalid relative time expression: %q", expr)
	}

	switch unit {
	case "s":
		return now.Add(-time.Duration(n) * time.Second), nil
	case "m":
		return now.Add(-time.Duration(n) * time.Minute), nil
	case "h":
		return now.Add(-time.Duration(n) * time.Hour), nil
	case "d":
		return now.Add(-time.Duration(n) * 24 * time.Hour), nil
	}
	return time.Time{}, fmt.Errorf("apm: unknown time unit %q in expression: %q", unit, expr)
}

func FormatISO(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
