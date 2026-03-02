package incidents

func SeverityFromPriority(priority, confidence, dq float64) string {
	if dq >= 0.60 || confidence < 0.35 {
		return "Data Unreliable"
	}
	if priority >= 97 {
		return "Critical"
	}
	if priority >= 90 {
		return "High Risk"
	}
	if priority >= 70 {
		return "Elevated"
	}
	return "Stable"
}
