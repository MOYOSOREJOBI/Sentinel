package logic

import "math"

func FeatureLogReturn(price float64) float64 { return math.Log(price / 100.0) }
func Severity(score float64) string {
	if score > 0.8 {
		return "critical"
	}
	if score > 0.6 {
		return "high"
	}
	if score > 0.4 {
		return "medium"
	}
	return "low"
}

type ModelState string

const (
	Draft    ModelState = "draft"
	Approved ModelState = "approved"
	Deployed ModelState = "deployed"
)

func Transition(s ModelState, evt string) ModelState {
	if s == Draft && evt == "approve" {
		return Approved
	}
	if s == Approved && evt == "deploy" {
		return Deployed
	}
	if evt == "rollback" {
		return Approved
	}
	return s
}
