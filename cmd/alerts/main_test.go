package main

import "testing"

func TestIsAllowedCaseTransition(t *testing.T) {
	cases := []struct {
		current string
		next    string
		ok      bool
	}{
		{"open", "investigating", true},
		{"open", "escalated", true},
		{"investigating", "closed", true},
		{"escalated", "open", false},
		{"closed", "open", false},
		{"closed", "closed", true},
	}
	for _, tc := range cases {
		if got := isAllowedCaseTransition(tc.current, tc.next); got != tc.ok {
			t.Fatalf("transition %s -> %s = %v, want %v", tc.current, tc.next, got, tc.ok)
		}
	}
}
