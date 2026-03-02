import json
from pathlib import Path

import app


def test_active_model_paths_from_config_file(tmp_path, monkeypatch):
    cfg = {
        "anomaly_model_path": "/models/anomaly.pkl",
        "escalation_model_path": "/models/escalation.pkl",
    }
    p = tmp_path / "active.json"
    p.write_text(json.dumps(cfg), encoding="utf-8")
    monkeypatch.setenv("ACTIVE_MODELS_PATH", str(p))
    monkeypatch.setenv("ANOMALY_MODEL_PATH", "")
    monkeypatch.setenv("ESCALATION_MODEL_PATH", "")

    anomaly_path, escalation_path = app._active_model_paths()

    assert anomaly_path == "/models/anomaly.pkl"
    assert escalation_path == "/models/escalation.pkl"


def test_active_model_paths_fall_back_to_env(monkeypatch):
    monkeypatch.setenv("ACTIVE_MODELS_PATH", "/does/not/exist.json")
    monkeypatch.setenv("ANOMALY_MODEL_PATH", "/env/anomaly.pkl")
    monkeypatch.setenv("ESCALATION_MODEL_PATH", "/env/escalation.pkl")

    anomaly_path, escalation_path = app._active_model_paths()

    assert anomaly_path == "/env/anomaly.pkl"
    assert escalation_path == "/env/escalation.pkl"
