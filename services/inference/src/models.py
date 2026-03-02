from __future__ import annotations
import hashlib
import json
import pickle
from pathlib import Path

DEFAULT_MODELS_DIR = Path(__file__).resolve().parent.parent / "models"


class LoadedModel:
    def __init__(self, model=None, version="fallback_v1", artifact_hash="fallback", degraded=True):
        self.model = model
        self.version = version
        self.artifact_hash = artifact_hash
        self.degraded = degraded


def _sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def default_model_paths() -> tuple[str, str]:
    return str(DEFAULT_MODELS_DIR / "anomaly.pkl"), str(DEFAULT_MODELS_DIR / "escalation.pkl")


def default_metadata_path() -> str:
    return str(DEFAULT_MODELS_DIR / "metadata.json")


def load_pickle_model(path: str, fallback_version: str) -> LoadedModel:
    p = Path(path)
    if not path or (not p.exists()) or p.is_dir():
        return LoadedModel(model=None, version=fallback_version, artifact_hash="missing", degraded=True)
    with p.open("rb") as f:
        model = pickle.load(f)
    return LoadedModel(model=model, version=p.stem, artifact_hash=_sha256_file(p), degraded=False)


def load_bundle_metadata(path: str = "") -> dict:
    p = Path(path or default_metadata_path())
    base = {
        "model_version": "fallback_v1",
        "trained_at": "",
        "dataset_window": "",
        "features_version": "v2",
        "artifact_hash": "",
        "fallback_mode": True,
        "calibration_version": "fallback",
        "metrics": {
            "auc": 0.0,
            "brier": 1.0,
            "pr_auc": 0.0,
            "anomaly_stability": 0.0,
        },
    }
    if not p.exists() or p.is_dir():
        return base
    try:
        loaded = json.loads(p.read_text(encoding="utf-8"))
    except Exception:
        return base
    if not isinstance(loaded, dict):
        return base
    merged = dict(base)
    merged.update({k: v for k, v in loaded.items() if k != "metrics"})
    metrics = dict(base["metrics"])
    if isinstance(loaded.get("metrics"), dict):
        metrics.update(loaded["metrics"])
    merged["metrics"] = metrics
    return merged
