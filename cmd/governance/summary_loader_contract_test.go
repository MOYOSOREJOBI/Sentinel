package main

import (
	"os"
	"strings"
	"testing"
)

func TestGovernanceSummaryLoadersHandleUUIDAndNullableErrorMessage(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	src := string(b)
	if !strings.Contains(src, "var id string") {
		t.Fatalf("expected deployment loader to scan UUID ids as strings")
	}
	if !strings.Contains(src, "coalesce(rj.error_message,'')") {
		t.Fatalf("expected replay job loader to coalesce nullable error_message")
	}
}
