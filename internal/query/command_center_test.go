package query

import "testing"

func TestBuildCommandCenterSummaryReturnsRealCountryRollups(t *testing.T) {
	rows := []QueueRow{
		{ID: 1, Symbol: "CONS-041", SeverityBand: "critical", PriorityScore: 0.92, CompositeRisk: 0.92, CountryISO2: "JP", CountryCode: "JP", CountryName: "Japan", Industry: "Retail"},
		{ID: 2, Symbol: "REIT-021", SeverityBand: "high", PriorityScore: 0.76, CompositeRisk: 0.76, CountryISO2: "AU", CountryCode: "AU", CountryName: "Australia", Industry: "REITs"},
		{ID: 3, Symbol: "UTIL-071", SeverityBand: "elevated", PriorityScore: 0.48, CompositeRisk: 0.48, CountryISO2: "GB", CountryCode: "GB", CountryName: "United Kingdom", Industry: "Utilities"},
		{ID: 4, Symbol: "TECH-001", SeverityBand: "stable", PriorityScore: 0.18, CompositeRisk: 0.18, CountryISO2: "US", CountryCode: "US", CountryName: "United States", Industry: "Software"},
		{ID: 5, Symbol: "UNKNOWN", SeverityBand: "stable", PriorityScore: 0.12, CompositeRisk: 0.12, CountryISO2: "XX", CountryCode: "XX", Industry: ""},
	}

	out := buildCommandCenterSummary("24h", rows)

	topCountries, ok := out["topCountries"].([]map[string]any)
	if !ok {
		t.Fatalf("topCountries missing or wrong type: %T", out["topCountries"])
	}
	if len(topCountries) == 0 {
		t.Fatalf("expected at least one real country rollup")
	}
	if topCountries[0]["countryCode"] == "XX" {
		t.Fatalf("expected placeholder country codes to be filtered out: %#v", topCountries)
	}
	if topCountries[0]["countryName"] == "" {
		t.Fatalf("expected real country names in rollups: %#v", topCountries)
	}
	if topCountries[0]["countryIso2"] == "" {
		t.Fatalf("expected canonical iso2 field in rollups: %#v", topCountries)
	}

	severity, ok := out["severityBreakdown"].(map[string]int)
	if !ok {
		t.Fatalf("severityBreakdown missing or wrong type: %T", out["severityBreakdown"])
	}
	if severity["critical"] != 1 || severity["high"] != 1 || severity["elevated"] != 1 || severity["stable"] != 2 {
		t.Fatalf("unexpected severity breakdown: %#v", severity)
	}

	histogram, ok := out["compositeHistogram"].([]int)
	if !ok {
		t.Fatalf("compositeHistogram missing or wrong type: %T", out["compositeHistogram"])
	}
	if len(histogram) != 5 {
		t.Fatalf("expected 5 histogram bins, got %d", len(histogram))
	}
	if histogram[0] == 0 || histogram[2] == 0 || histogram[3] == 0 || histogram[4] == 0 {
		t.Fatalf("expected a spread across bins, got %#v", histogram)
	}
}
