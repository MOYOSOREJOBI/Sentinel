from dataclasses import asdict, dataclass
from typing import Any


@dataclass
class ScoreOutput:
    symbol: str
    raw_anomaly_score: float
    normalized_anomaly_score: float
    escalation_probability: float
    confidence: float
    expected_severity_band: str
    priority_score: float
    recommended_action: str
    composite_risk: float
    safety_level: str
    top_drivers: list[dict[str, Any]]
    explanation_text: str
    feature_snapshot_hash: str
    feature_set_version: str
    model_version: str
    artifact_hash: str
    fallback_mode: bool
    dq_penalty: float

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


def normalize_output(payload: dict[str, Any]) -> dict[str, Any]:
    return payload
