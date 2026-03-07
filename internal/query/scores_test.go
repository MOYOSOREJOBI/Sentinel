package query

import (
	"strings"
	"testing"
)

func TestScoreSeriesExpressionsForRollupsUseAggregateColumnNames(t *testing.T) {
	selectCols, orderCol := scoreSeriesExpressions("scores_1h", "bucket")

	if strings.Contains(selectCols, "normalized_anomaly_score") {
		t.Fatalf("aggregate score query must not reference raw-only normalized_anomaly_score columns: %s", selectCols)
	}
	if !strings.Contains(selectCols, "COALESCE(score_norm, 0)") {
		t.Fatalf("expected aggregate score query to use rollup columns: %s", selectCols)
	}
	if orderCol != "COALESCE(produced_at, ts)" {
		t.Fatalf("unexpected aggregate order column: %s", orderCol)
	}
}
