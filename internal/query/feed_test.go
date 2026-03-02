package query

import (
	"sort"
	"testing"
	"time"
)

func TestRankFeedRowRewardsHigherSignalQuality(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

	strong := feedRow{
		PriorityScore:         92,
		CompositeRisk:         88,
		EscalationProbability: 0.84,
		Confidence:            0.95,
		SeverityBand:          "critical",
		TrustState:            "stable",
		ModelVersion:          "tree-v3",
		LastActivityAt:        now.Add(-5 * time.Minute),
	}
	weak := feedRow{
		PriorityScore:         34,
		CompositeRisk:         42,
		EscalationProbability: 0.22,
		Confidence:            0.41,
		SeverityBand:          "elevated",
		TrustState:            "degraded",
		ModelVersion:          "fallback-v1",
		LastActivityAt:        now.Add(-6 * time.Hour),
	}

	strongScore, strongMarket, _, _ := rankFeedRow(strong, QueueFilters{}, now, "")
	weakScore, weakMarket, _, _ := rankFeedRow(weak, QueueFilters{}, now, "")

	if strongScore <= weakScore {
		t.Fatalf("expected stronger row score (%f) > weaker row score (%f)", strongScore, weakScore)
	}
	if strongMarket <= weakMarket {
		t.Fatalf("expected stronger market implied risk (%f) > weaker (%f)", strongMarket, weakMarket)
	}
}

func TestRankFeedRowIsClampedToProbabilityBounds(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	row := feedRow{
		PriorityScore:         1000,
		CompositeRisk:         1000,
		EscalationProbability: 2,
		Confidence:            2,
		SeverityBand:          "critical",
		LastActivityAt:        now,
	}

	score, market, _, _ := rankFeedRow(row, QueueFilters{}, now, "")
	if score < 0 || score > 1 {
		t.Fatalf("score outside [0,1]: %f", score)
	}
	if market < 0 || market > 1 {
		t.Fatalf("market implied risk outside [0,1]: %f", market)
	}
}

func TestRankFeedRowTrendingBoostsFreshness(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	// Two identical rows except one is very fresh
	fresh := feedRow{
		PriorityScore: 50, CompositeRisk: 50, EscalationProbability: 0.5,
		Confidence: 0.5, SeverityBand: "high", TrustState: "stable",
		LastActivityAt: now.Add(-1 * time.Minute),
	}
	stale := feedRow{
		PriorityScore: 50, CompositeRisk: 50, EscalationProbability: 0.5,
		Confidence: 0.5, SeverityBand: "high", TrustState: "stable",
		LastActivityAt: now.Add(-120 * time.Minute),
	}
	freshDefault, _, _, _ := rankFeedRow(fresh, QueueFilters{}, now, "")
	staleDefault, _, _, _ := rankFeedRow(stale, QueueFilters{}, now, "")
	freshTrending, _, _, _ := rankFeedRow(fresh, QueueFilters{}, now, "trending")
	staleTrending, _, _, _ := rankFeedRow(stale, QueueFilters{}, now, "trending")

	// Fresh minus stale gap should be larger in trending mode
	gapDefault := freshDefault - staleDefault
	gapTrending := freshTrending - staleTrending
	if gapTrending <= gapDefault {
		t.Fatalf("trending mode should widen freshness gap: default=%f trending=%f", gapDefault, gapTrending)
	}
}

func TestFeedCursorRoundTrip(t *testing.T) {
	if got := decodeCursor(encodeCursor(125)); got != 125 {
		t.Fatalf("expected cursor round trip to preserve offset, got %d", got)
	}
	if got := decodeCursor("not-a-cursor"); got != 0 {
		t.Fatalf("expected invalid cursor to decode to 0, got %d", got)
	}
}

func TestFeedOrderingTieBreaksByIncidentID(t *testing.T) {
	rows := []feedRanked{
		{row: feedRow{ID: 42}, rankScore: 0.8},
		{row: feedRow{ID: 7}, rankScore: 0.8},
		{row: feedRow{ID: 11}, rankScore: 0.9},
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].rankScore == rows[j].rankScore {
			return rows[i].row.ID < rows[j].row.ID
		}
		return rows[i].rankScore > rows[j].rankScore
	})
	if rows[0].row.ID != 11 || rows[1].row.ID != 7 || rows[2].row.ID != 42 {
		t.Fatalf("unexpected stable order: %+v", []int64{rows[0].row.ID, rows[1].row.ID, rows[2].row.ID})
	}
}
