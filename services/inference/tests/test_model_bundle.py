import json
import pickle

import app
from src.simple_models import CalibratedLogisticModel, RobustAnomalyModel


def test_model_bundle_flips_out_of_fallback_when_files_exist(tmp_path, monkeypatch):
    anomaly_path = tmp_path / "anomaly.pkl"
    escalation_path = tmp_path / "escalation.pkl"
    metadata_path = tmp_path / "metadata.json"

    with anomaly_path.open("wb") as f:
        pickle.dump(RobustAnomalyModel([0.0, 0.0], [1.0, 1.0]), f)
    with escalation_path.open("wb") as f:
        pickle.dump(CalibratedLogisticModel([0.0, 0.0], [1.0, 1.0], [0.5, -0.25], 0.1), f)
    metadata_path.write_text(
        json.dumps(
            {
                "model_version": "test-quant-v1",
                "trained_at": "2026-03-01T00:00:00Z",
                "artifact_hash": "bundle-hash",
                "fallback_mode": False,
                "metrics": {"auc": 0.71, "brier": 0.19, "pr_auc": 0.43, "anomaly_stability": 0.82},
            }
        ),
        encoding="utf-8",
    )

    monkeypatch.setenv("ANOMALY_MODEL_PATH", str(anomaly_path))
    monkeypatch.setenv("ESCALATION_MODEL_PATH", str(escalation_path))
    monkeypatch.setenv("MODEL_METADATA_PATH", str(metadata_path))
    monkeypatch.setenv("ACTIVE_MODELS_PATH", "")

    bundle = app.init_model_bundle(force=True)

    assert bundle["fallback_mode"] is False
    assert bundle["model_version"] == "test-quant-v1"
    assert bundle["model_artifact_hash"] == "bundle-hash"
