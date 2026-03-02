package main

import (
	"math"
	"testing"
	"time"
)

func TestComputeFeaturesComputesHorizonReturns(t *testing.T) {
	st := newSymbolState()
	base := time.Now().UTC()
	for i := 0; i < 300; i++ {
		price := 100.0 + float64(i)
		computeFeatures(st, price, 1000, base.Add(time.Duration(i)*time.Second))
	}
	payload, _ := computeFeatures(st, 500.0, 1000, base.Add(301*time.Second))

	ret1s := payload["ret_1s"]
	ret1m := payload["ret_1m"]
	ret5m := payload["ret_5m"]
	if math.Abs(ret1s-((500.0/399.0)-1)) > 1e-9 {
		t.Fatalf("unexpected ret_1s: %f", ret1s)
	}
	if math.Abs(ret1m-((500.0/340.0)-1)) > 1e-9 {
		t.Fatalf("unexpected ret_1m: %f", ret1m)
	}
	if math.Abs(ret5m-((500.0/100.0)-1)) > 1e-9 {
		t.Fatalf("unexpected ret_5m: %f", ret5m)
	}
}
