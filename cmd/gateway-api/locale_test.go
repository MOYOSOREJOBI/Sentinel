package main

import "testing"

func TestNormalizePreferredLocale(t *testing.T) {
	if got := normalizePreferredLocale("ar"); got != "ar" {
		t.Fatalf("expected ar, got %q", got)
	}
	if got := normalizePreferredLocale("zh"); got != "zh-Hans" {
		t.Fatalf("expected zh-Hans, got %q", got)
	}
	if got := normalizePreferredLocale("bad-value"); got != "en" {
		t.Fatalf("expected en fallback, got %q", got)
	}
}
