package main

import "testing"

func TestHealthcheckAddr(t *testing.T) {
	if got := healthcheckAddr("", ":8086"); got != ":8086" {
		t.Fatalf("healthcheckAddr empty=%q want :8086", got)
	}
	if got := healthcheckAddr(":9000", ":8086"); got != ":9000" {
		t.Fatalf("healthcheckAddr explicit=%q want :9000", got)
	}
}
