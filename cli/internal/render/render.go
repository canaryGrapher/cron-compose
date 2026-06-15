// Package render holds tiny formatting helpers shared across commands.
package render

import "time"

// ShortID returns the first 8 chars of a ULID for compact tables.
func ShortID(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

// Time formats an optional RFC 3339 timestamp pointer. Empty when nil.
func Time(p *string) string {
	if p == nil || *p == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, *p)
	if err != nil {
		return *p
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
