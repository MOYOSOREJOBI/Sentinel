package query

import (
	"strings"
	"testing"
	"time"
)

func TestWhereClauseCustomWindowUsesBetween(t *testing.T) {
	f := QueueFilters{
		Window:      "custom",
		From:        time.Date(2025, 1, 2, 1, 0, 0, 0, time.UTC),
		To:          time.Date(2025, 1, 2, 2, 0, 0, 0, time.UTC),
		CountryCode: "US",
		Region:      "NA",
		Sector:      "Technology",
		Industry:    "Software",
		Venue:       "XNAS",
		Symbol:      "AAPL",
	}
	where, args := whereClause(f)
	if !strings.Contains(where, "i.last_activity_at BETWEEN $1 AND $2") {
		t.Fatalf("expected custom between clause, got %q", where)
	}
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}
}

func TestWhereClauseAddsSearchQuery(t *testing.T) {
	f := QueueFilters{Window: "24h", Q: "bank"}
	where, args := whereClause(f)
	if !strings.Contains(where, "lower(i.primary_symbol)") {
		t.Fatalf("expected q search clause, got %q", where)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestWhereClauseNonCustomIgnoresPartialRange(t *testing.T) {
	f := QueueFilters{Window: "24h", From: time.Now().UTC(), CountryCode: "DE"}
	where, args := whereClause(f)
	if strings.Contains(where, "BETWEEN") {
		t.Fatalf("did not expect between clause in non-custom window")
	}
	if len(args) != 1 {
		t.Fatalf("expected only country filter arg, got %d", len(args))
	}
}

func TestWhereClauseCustomWindowSupportsFromOnly(t *testing.T) {
	f := QueueFilters{Window: "custom", From: time.Date(2025, 1, 2, 1, 0, 0, 0, time.UTC)}
	where, args := whereClause(f)
	if !strings.Contains(where, "i.last_activity_at >= $1") {
		t.Fatalf("expected lower-bound custom clause, got %q", where)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestWhereClauseCustomWindowSupportsToOnly(t *testing.T) {
	f := QueueFilters{Window: "custom", To: time.Date(2025, 1, 2, 2, 0, 0, 0, time.UTC)}
	where, args := whereClause(f)
	if !strings.Contains(where, "i.last_activity_at <= $1") {
		t.Fatalf("expected upper-bound custom clause, got %q", where)
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestWhereClauseCustomWindowSwapsInvertedRange(t *testing.T) {
	f := QueueFilters{
		Window: "custom",
		From:   time.Date(2025, 1, 2, 5, 0, 0, 0, time.UTC),
		To:     time.Date(2025, 1, 2, 1, 0, 0, 0, time.UTC),
	}
	_, args := whereClause(f)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	from, ok1 := args[0].(time.Time)
	to, ok2 := args[1].(time.Time)
	if !ok1 || !ok2 {
		t.Fatalf("expected time args")
	}
	if from.After(to) {
		t.Fatalf("expected normalized from<=to; got from=%s to=%s", from, to)
	}
}
