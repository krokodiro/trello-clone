package handler

import (
	"testing"
	"time"
)

func TestParseDueDate(t *testing.T) {
	rfc := "2026-06-26T12:00:00Z"
	got := parseDueDate(&rfc)
	if got == nil {
		t.Fatal("expected parsed RFC3339 date")
	}
	if got.UTC().Format(time.RFC3339) != rfc {
		t.Fatalf("got %v", got)
	}

	day := "2026-06-26"
	got = parseDueDate(&day)
	if got == nil {
		t.Fatal("expected parsed date")
	}

	empty := ""
	if parseDueDate(&empty) != nil {
		t.Fatal("expected nil for empty string")
	}
	if parseDueDate(nil) != nil {
		t.Fatal("expected nil for nil")
	}

	bad := "not-a-date"
	if parseDueDate(&bad) != nil {
		t.Fatal("expected nil for invalid date")
	}
}
