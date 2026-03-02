from __future__ import annotations

import json
import math
import pickle
import random
import statistics
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

SERVICE_DIR = Path(__file__).resolve().parents[1]
REPO_ROOT = Path(__file__).resolve().parents[3]
if str(SERVICE_DIR) not in sys.path:
    sys.path.insert(0, str(SERVICE_DIR))

from src.artifacts import stable_sha256
from src.feature_schema import FEATURE_COLUMNS
from src.simple_models import CalibratedLogisticModel, RobustAnomalyModel

MODELS_DIR = SERVICE_DIR / "models"
ANOMALY_PATH = MODELS_DIR / "anomaly.pkl"
ESCALATION_PATH = MODELS_DIR / "escalation.pkl"
METADATA_PATH = MODELS_DIR / "metadata.json"
MIN_AUC = 0.60
MAX_BRIER = 0.25

TRAINING_SQL = """
WITH recent_features AS (
  SELECT symbol, ts, payload
  FROM features
  WHERE ts >= now() - interval '30 days'
  ORDER BY ts DESC
  LIMIT 4000
),
labeled AS (
  SELECT
    COALESCE((f.payload->'payload'->>'anomaly_density_300')::double precision, 0) AS anomaly_density_300,
    COALESCE((f.payload->'payload'->>'dq_penalty')::double precision, 0) AS dq_penalty,
    COALESCE((f.payload->'payload'->>'duplicate_ratio')::double precision, 0) AS duplicate_ratio,
    COALESCE((f.payload->'payload'->>'ewma_var_300')::double precision, 0) AS ewma_var_300,
    COALESCE((f.payload->'payload'->>'ewma_var_60')::double precision, 0) AS ewma_var_60,
    COALESCE((f.payload->'payload'->>'ewma_vol_300')::double precision, 0) AS ewma_vol_300,
    COALESCE((f.payload->'payload'->>'ewma_vol_60')::double precision, 0) AS ewma_vol_60,
    COALESCE((f.payload->'payload'->>'late_ratio')::double precision, 0) AS late_ratio,
    COALESCE((f.payload->'payload'->>'log_ret_1m')::double precision, 0) AS log_ret_1m,
    COALESCE((f.payload->'payload'->>'log_ret_1s')::double precision, 0) AS log_ret_1s,
    COALESCE((f.payload->'payload'->>'log_ret_5m')::double precision, 0) AS log_ret_5m,
    COALESCE((f.payload->'payload'->>'mean_ret_300')::double precision, 0) AS mean_ret_300,
    COALESCE((f.payload->'payload'->>'mean_ret_60')::double precision, 0) AS mean_ret_60,
    COALESCE((f.payload->'payload'->>'mean_vol_60')::double precision, 0) AS mean_vol_60,
    COALESCE((f.payload->'payload'->>'missingness_ratio')::double precision, 0) AS missingness_ratio,
    COALESCE((f.payload->'payload'->>'out_of_order_ratio')::double precision, 0) AS out_of_order_ratio,
    COALESCE((f.payload->'payload'->>'realized_var_300')::double precision, 0) AS realized_var_300,
    COALESCE((f.payload->'payload'->>'realized_vol_300')::double precision, 0) AS realized_vol_300,
    COALESCE((f.payload->'payload'->>'ret_1m')::double precision, 0) AS ret_1m,
    COALESCE((f.payload->'payload'->>'ret_1s')::double precision, 0) AS ret_1s,
    COALESCE((f.payload->'payload'->>'ret_5m')::double precision, 0) AS ret_5m,
    COALESCE((f.payload->'payload'->>'staleness_score')::double precision, 0) AS staleness_score,
    COALESCE((f.payload->'payload'->>'std_ret_300')::double precision, 0) AS std_ret_300,
    COALESCE((f.payload->'payload'->>'std_ret_60')::double precision, 0) AS std_ret_60,
    COALESCE((f.payload->'payload'->>'std_vol_60')::double precision, 0) AS std_vol_60,
    COALESCE((f.payload->'payload'->>'tick_gap_s')::double precision, 0) AS tick_gap_s,
    COALESCE((f.payload->'payload'->>'volume_ratio_60')::double precision, 0) AS volume_ratio_60,
    COALESCE((f.payload->'payload'->>'volume_surprise_60')::double precision, 0) AS volume_surprise_60,
    COALESCE((f.payload->'payload'->>'z_ret_300')::double precision, 0) AS z_ret_300,
    COALESCE((f.payload->'payload'->>'z_ret_60')::double precision, 0) AS z_ret_60,
    CASE
      WHEN EXISTS (
        SELECT 1
        FROM alerts a
        JOIN cases c ON c.incident_id = a.incident_id
        WHERE a.score_id = s.id
      ) THEN 1
      WHEN COALESCE(s.severity, 'stable') IN ('high', 'critical') THEN 1
      ELSE 0
    END AS escalation_label
  FROM recent_features f
  LEFT JOIN LATERAL (
    SELECT id, severity
    FROM scores s
    WHERE s.symbol = f.symbol
      AND s.ts BETWEEN f.ts - interval '2 minutes' AND f.ts + interval '2 minutes'
    ORDER BY ABS(EXTRACT(EPOCH FROM (s.ts - f.ts))) ASC
    LIMIT 1
  ) s ON true
)
SELECT
  anomaly_density_300, dq_penalty, duplicate_ratio, ewma_var_300, ewma_var_60,
  ewma_vol_300, ewma_vol_60, late_ratio, log_ret_1m, log_ret_1s, log_ret_5m,
  mean_ret_300, mean_ret_60, mean_vol_60, missingness_ratio, out_of_order_ratio,
  realized_var_300, realized_vol_300, ret_1m, ret_1s, ret_5m, staleness_score,
  std_ret_300, std_ret_60, std_vol_60, tick_gap_s, volume_ratio_60,
  volume_surprise_60, z_ret_300, z_ret_60, escalation_label
FROM labeled
"""


def _compose_prefix() -> list[str]:
    return ["docker", "compose", "-f", str(REPO_ROOT / "deploy/docker/docker-compose.yml")]


def fetch_training_rows() -> list[tuple[list[float], int]]:
    cmd = _compose_prefix() + [
        "exec",
        "-T",
        "postgres",
        "psql",
        "-U",
        "sentinel",
        "-d",
        "sentinel",
        "-At",
        "-F",
        "\t",
        "-c",
        TRAINING_SQL,
    ]
    proc = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or "psql query failed")
    rows: list[tuple[list[float], int]] = []
    for line in proc.stdout.splitlines():
        if not line.strip():
            continue
        parts = line.split("\t")
        if len(parts) != len(FEATURE_COLUMNS) + 1:
            continue
        vec = [float(parts[i] or 0.0) for i in range(len(FEATURE_COLUMNS))]
        label = int(parts[-1] or 0)
        rows.append((vec, label))
    random.Random(731103).shuffle(rows)
    return rows


def split_rows(rows: list[tuple[list[float], int]]) -> tuple[list[tuple[list[float], int]], list[tuple[list[float], int]]]:
    if len(rows) < 500:
        raise RuntimeError(f"not enough joined training rows for a real model bundle: {len(rows)}")
    split_at = max(int(len(rows) * 0.8), 1)
    train = rows[:split_at]
    valid = rows[split_at:] or rows[-max(len(rows) // 5, 1):]
    return train, valid


def _column(values: list[list[float]], idx: int) -> list[float]:
    return [row[idx] for row in values]


def _median(values: list[float]) -> float:
    return float(statistics.median(values)) if values else 0.0


def _mean(values: list[float]) -> float:
    return float(sum(values) / max(len(values), 1))


def _std(values: list[float], mean_value: float) -> float:
    if len(values) < 2:
        return 1.0
    variance = sum((value-mean_value) ** 2 for value in values) / (len(values) - 1)
    return math.sqrt(max(variance, 1e-9))


def train_anomaly_model(train_rows: list[list[float]]) -> tuple[RobustAnomalyModel, float]:
    medians: list[float] = []
    scales: list[float] = []
    for idx in range(len(FEATURE_COLUMNS)):
        column = _column(train_rows, idx)
        med = _median(column)
        abs_dev = [abs(value-med) for value in column]
        mad = _median(abs_dev)
        scale = max(1.4826 * mad, 1e-4)
        medians.append(med)
        scales.append(scale)
    model = RobustAnomalyModel(medians, scales)
    anomaly_scores = [(-score + 0.10) / 0.60 for score in model.decision_function(train_rows)]
    stability = max(0.0, 1.0-min(statistics.pstdev(anomaly_scores) if len(anomaly_scores) > 1 else 0.0, 1.0))
    return model, stability


def _sigmoid(value: float) -> float:
    if value >= 0:
        z = math.exp(-value)
        return 1.0 / (1.0 + z)
    z = math.exp(value)
    return z / (1.0 + z)


def _standardize(train_rows: list[list[float]]) -> tuple[list[float], list[float], list[list[float]]]:
    means: list[float] = []
    scales: list[float] = []
    standardized: list[list[float]] = []
    for idx in range(len(FEATURE_COLUMNS)):
        column = _column(train_rows, idx)
        mean_value = _mean(column)
        scale = max(_std(column, mean_value), 1e-4)
        means.append(mean_value)
        scales.append(scale)
    for row in train_rows:
        standardized.append([(row[i]-means[i]) / scales[i] for i in range(len(FEATURE_COLUMNS))])
    return means, scales, standardized


def _fit_logistic(z_rows: list[list[float]], labels: list[int], epochs: int = 400, lr: float = 0.05) -> tuple[list[float], float]:
    positives = sum(labels)
    total = max(len(labels), 1)
    base_rate = min(max(positives / total, 1e-4), 1.0-1e-4)
    bias = math.log(base_rate / (1.0-base_rate))
    weights = [0.0 for _ in range(len(z_rows[0]))]
    for _ in range(epochs):
        grad_w = [0.0 for _ in weights]
        grad_b = 0.0
        for row, label in zip(z_rows, labels):
            logit = bias + sum(weight * value for weight, value in zip(weights, row))
            prob = min(max(_sigmoid(logit), 1e-6), 1.0-1e-6)
            err = prob - float(label)
            grad_b += err
            for idx, value in enumerate(row):
                grad_w[idx] += err * value
        scale = 1.0 / total
        bias -= lr * grad_b * scale
        for idx in range(len(weights)):
            weights[idx] -= lr * grad_w[idx] * scale
    return weights, bias


def _fit_platt(logits: list[float], labels: list[int], epochs: int = 300, lr: float = 0.03) -> tuple[float, float]:
    a = 1.0
    b = 0.0
    total = max(len(logits), 1)
    for _ in range(epochs):
        grad_a = 0.0
        grad_b = 0.0
        for logit, label in zip(logits, labels):
            prob = min(max(_sigmoid((a * logit) + b), 1e-6), 1.0-1e-6)
            err = prob - float(label)
            grad_a += err * logit
            grad_b += err
        a -= lr * grad_a / total
        b -= lr * grad_b / total
    return a, b


def train_escalation_model(train: list[tuple[list[float], int]], valid: list[tuple[list[float], int]]) -> tuple[CalibratedLogisticModel, dict]:
    train_rows = [row for row, _ in train]
    train_labels = [label for _, label in train]
    valid_rows = [row for row, _ in valid]
    valid_labels = [label for _, label in valid]
    if len(set(train_labels)) < 2 or len(set(valid_labels)) < 2:
        raise RuntimeError("training data needs both positive and negative escalation labels")
    means, scales, z_rows = _standardize(train_rows)
    weights, bias = _fit_logistic(z_rows, train_labels)
    valid_logits = []
    for row in valid_rows:
        valid_logits.append(bias + sum(weights[i] * ((row[i]-means[i]) / scales[i]) for i in range(len(FEATURE_COLUMNS))))
    calibration_a, calibration_b = _fit_platt(valid_logits, valid_labels)
    model = CalibratedLogisticModel(means, scales, weights, bias, calibration_a, calibration_b)
    probs = [pair[1] for pair in model.predict_proba(valid_rows)]
    metrics = {
        "auc": roc_auc(valid_labels, probs),
        "brier": brier_score(valid_labels, probs),
        "pr_auc": pr_auc(valid_labels, probs),
    }
    return model, metrics


def brier_score(labels: list[int], probs: list[float]) -> float:
    return sum((prob-float(label)) ** 2 for label, prob in zip(labels, probs)) / max(len(labels), 1)


def roc_auc(labels: list[int], probs: list[float]) -> float:
    paired = sorted(zip(probs, labels), key=lambda item: item[0])
    pos = sum(labels)
    neg = len(labels) - pos
    if pos == 0 or neg == 0:
        return 0.5
    rank_sum = 0.0
    for idx, (_, label) in enumerate(paired, start=1):
        if label == 1:
            rank_sum += idx
    return (rank_sum - (pos * (pos + 1) / 2.0)) / (pos * neg)


def pr_auc(labels: list[int], probs: list[float]) -> float:
    paired = sorted(zip(probs, labels), key=lambda item: item[0], reverse=True)
    positives = sum(labels)
    if positives == 0:
        return 0.0
    tp = 0
    fp = 0
    prev_recall = 0.0
    area = 0.0
    for _, label in paired:
        if label == 1:
            tp += 1
        else:
            fp += 1
        recall = tp / positives
        precision = tp / max(tp + fp, 1)
        area += (recall-prev_recall) * precision
        prev_recall = recall
    return area


def write_bundle(anomaly_model: RobustAnomalyModel, escalation_model: CalibratedLogisticModel, metrics: dict, anomaly_stability: float, training_rows: int) -> dict:
    MODELS_DIR.mkdir(parents=True, exist_ok=True)
    with ANOMALY_PATH.open("wb") as f:
        pickle.dump(anomaly_model, f)
    with ESCALATION_PATH.open("wb") as f:
        pickle.dump(escalation_model, f)

    trained_at = datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    anomaly_hash = stable_sha256([ANOMALY_PATH.read_bytes().hex()])
    escalation_hash = stable_sha256([ESCALATION_PATH.read_bytes().hex()])
    bundle_hash = stable_sha256([anomaly_hash, escalation_hash, trained_at, str(training_rows)])
    metadata = {
        "model_version": f"quant-v1-{trained_at[:10]}",
        "trained_at": trained_at,
        "dataset_window": "30d",
        "features_version": "v2",
        "artifact_hash": bundle_hash,
        "fallback_mode": False,
        "calibration_version": "platt_v1",
        "metrics": {
            "auc": round(float(metrics["auc"]), 6),
            "brier": round(float(metrics["brier"]), 6),
            "pr_auc": round(float(metrics["pr_auc"]), 6),
            "anomaly_stability": round(float(anomaly_stability), 6),
        },
    }
    METADATA_PATH.write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    return metadata


def main() -> int:
    rows = fetch_training_rows()
    train, valid = split_rows(rows)
    train_rows = [row for row, _ in train]
    anomaly_model, anomaly_stability = train_anomaly_model(train_rows)
    escalation_model, metrics = train_escalation_model(train, valid)
    print(json.dumps({"training_rows": len(train), "validation_rows": len(valid), "metrics": metrics, "anomaly_stability": anomaly_stability}, indent=2))
    if metrics["auc"] < MIN_AUC:
        raise RuntimeError(f"auc below threshold: {metrics['auc']:.4f} < {MIN_AUC:.2f}")
    if metrics["brier"] > MAX_BRIER:
        raise RuntimeError(f"brier above threshold: {metrics['brier']:.4f} > {MAX_BRIER:.2f}")
    metadata = write_bundle(anomaly_model, escalation_model, metrics, anomaly_stability, len(rows))
    print(json.dumps({"bundle": str(MODELS_DIR), "model_version": metadata["model_version"], "artifact_hash": metadata["artifact_hash"]}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
