from .scoring import clip


def safety_level_from_bounds(priority: float, dq_penalty: float, confidence: float) -> str:
    if dq_penalty >= 0.60 or confidence < 0.35:
        return "Data Unreliable"
    if priority >= 97:
        return "Critical"
    if priority >= 90:
        return "High Risk"
    if priority >= 70:
        return "Elevated"
    return "Stable"


def band_from_probability(p: float) -> str:
    if p >= 0.85:
        return "critical"
    if p >= 0.65:
        return "high"
    if p >= 0.40:
        return "elevated"
    return "stable"
