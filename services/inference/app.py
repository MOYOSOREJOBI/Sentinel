from __future__ import annotations

import json
import logging
import os
import signal
import sys
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

from flask import Flask, Response, jsonify
from kafka import KafkaConsumer, KafkaProducer
from prometheus_client import CONTENT_TYPE_LATEST, Counter, Gauge, Histogram, generate_latest

SERVICE_DIR = Path(__file__).resolve().parent
if str(SERVICE_DIR) not in sys.path:
    sys.path.insert(0, str(SERVICE_DIR))

from src.artifacts import bucket_rfc3339, snapshot_hash, stable_sha256
from src.explain import deterministic_explain
from src.fallbacks import safety_level_from_bounds
from src.feature_schema import vectorize_features
from src.models import default_metadata_path, default_model_paths, load_bundle_metadata, load_pickle_model
from src.scoring import composite_risk, priority_score, recommended_action, score_anomaly, score_escalation

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)
app = Flask(__name__)
BROKER = os.getenv("KAFKA_BROKER", "redpanda:9092")
CONSUMER_TOPIC = os.getenv("CONSUMER_TOPIC", "derived.features")
PRODUCER_TOPIC = os.getenv("PRODUCER_TOPIC", "derived.scores")
CONSUMER_GROUP = os.getenv("CONSUMER_GROUP", "inference")

MESSAGES_CONSUMED = Counter("inference_messages_consumed_total", "Total consumed feature messages")
MESSAGES_PRODUCED = Counter("inference_messages_produced_total", "Total produced score messages")
PROCESSING_ERRORS = Counter("inference_processing_errors_total", "Total inference processing errors", ["status"])
LOOP_RUNNING = Gauge("inference_loop_running", "Inference consumer loop running state (1/0)")
SCORING_LATENCY = Histogram(
    "inference_scoring_latency_seconds",
    "End-to-end scoring latency per message",
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5],
)
MODEL_LOADED = Gauge("inference_model_loaded", "Whether the model is loaded (1) or in fallback (0)", ["model_type"])
FALLBACK_MODE = Gauge("inference_fallback_mode", "Whether inference is currently using deterministic fallback scoring")
ESCALATION_AUC = Gauge("inference_escalation_auc", "Escalation model ROC AUC from the active bundle")
ESCALATION_BRIER = Gauge("inference_escalation_brier", "Escalation model Brier score from the active bundle")
ESCALATION_PR_AUC = Gauge("inference_escalation_pr_auc", "Escalation model PR-AUC from the active bundle")
ANOMALY_STABILITY = Gauge("inference_anomaly_stability_score", "Stability score for the active anomaly model bundle")

running = True
kafka_ready = False
model_bundle: dict | None = None
model_loaded = False
model_lock = threading.RLock()


def _active_model_paths() -> tuple[str, str]:
    default_anomaly, default_escalation = default_model_paths()
    active_cfg_path = os.getenv("ACTIVE_MODELS_PATH", "")
    if active_cfg_path:
        try:
            with open(active_cfg_path, "r", encoding="utf-8") as f:
                cfg = json.load(f)
            anomaly_path = str(cfg.get("anomaly_model_path", "")).strip()
            escalation_path = str(cfg.get("escalation_model_path", "")).strip()
            return anomaly_path or default_anomaly, escalation_path or default_escalation
        except Exception:
            logger.warning("failed to load active model config; using env model paths", exc_info=True)
    anomaly_path = os.getenv("ANOMALY_MODEL_PATH", "").strip()
    escalation_path = os.getenv("ESCALATION_MODEL_PATH", "").strip()
    return anomaly_path or default_anomaly, escalation_path or default_escalation


def _active_metadata_path() -> str:
    active_cfg_path = os.getenv("ACTIVE_MODELS_PATH", "")
    if active_cfg_path:
        try:
            with open(active_cfg_path, "r", encoding="utf-8") as f:
                cfg = json.load(f)
            metadata_path = str(cfg.get("metadata_path", "")).strip()
            if metadata_path:
                return metadata_path
        except Exception:
            logger.warning("failed to load active model metadata path; using env defaults", exc_info=True)
    return os.getenv("MODEL_METADATA_PATH", "").strip() or default_metadata_path()


def _bundle_model_version(bundle: dict) -> str:
    metadata = bundle.get("metadata", {})
    explicit = str(metadata.get("model_version", "")).strip()
    if explicit:
        return explicit
    anomaly = bundle["anomaly"]
    escalation = bundle["escalation"]
    return f"{getattr(anomaly, 'version', 'anomaly_fallback_v1')},{getattr(escalation, 'version', 'escalation_fallback_v1')}"


def _bundle_artifact_hash(bundle: dict) -> str:
    metadata = bundle.get("metadata", {})
    explicit = str(metadata.get("artifact_hash", "")).strip()
    if explicit:
        return explicit
    anomaly = bundle["anomaly"]
    escalation = bundle["escalation"]
    return stable_sha256(
        [
            str(getattr(anomaly, "artifact_hash", "missing")),
            str(getattr(escalation, "artifact_hash", "missing")),
        ]
    )


def _bundle_fallback_mode(bundle: dict) -> bool:
    anomaly = bundle["anomaly"]
    escalation = bundle["escalation"]
    return bool(getattr(anomaly, "degraded", getattr(anomaly, "model", None) is None)) or bool(
        getattr(escalation, "degraded", getattr(escalation, "model", None) is None)
    )


def _sync_model_metrics(bundle: dict | None) -> None:
    if not bundle:
        MODEL_LOADED.labels(model_type="anomaly").set(0)
        MODEL_LOADED.labels(model_type="escalation").set(0)
        FALLBACK_MODE.set(1)
        ESCALATION_AUC.set(0)
        ESCALATION_BRIER.set(1)
        ESCALATION_PR_AUC.set(0)
        ANOMALY_STABILITY.set(0)
        return

    anomaly = bundle["anomaly"]
    escalation = bundle["escalation"]
    MODEL_LOADED.labels(model_type="anomaly").set(0 if getattr(anomaly, "degraded", True) else 1)
    MODEL_LOADED.labels(model_type="escalation").set(0 if getattr(escalation, "degraded", True) else 1)
    FALLBACK_MODE.set(1 if _bundle_fallback_mode(bundle) else 0)
    metrics = bundle.get("metadata", {}).get("metrics", {})
    ESCALATION_AUC.set(float(metrics.get("auc", 0.0) or 0.0))
    ESCALATION_BRIER.set(float(metrics.get("brier", 1.0) or 1.0))
    ESCALATION_PR_AUC.set(float(metrics.get("pr_auc", 0.0) or 0.0))
    ANOMALY_STABILITY.set(float(metrics.get("anomaly_stability", 0.0) or 0.0))


def load_models() -> dict:
    anomaly_path, escalation_path = _active_model_paths()
    metadata = load_bundle_metadata(_active_metadata_path())
    anomaly = load_pickle_model(anomaly_path, "anomaly_fallback_v1")
    escalation = load_pickle_model(escalation_path, "escalation_fallback_v1")
    bundle = {
        "anomaly": anomaly,
        "escalation": escalation,
        "metadata": metadata,
    }
    bundle["model_version"] = _bundle_model_version(bundle)
    bundle["model_artifact_hash"] = _bundle_artifact_hash(bundle)
    bundle["fallback_mode"] = _bundle_fallback_mode(bundle)
    bundle["trained_at"] = str(metadata.get("trained_at", "") or "")
    bundle["calibration_version"] = str(metadata.get("calibration_version", "") or "fallback")
    return bundle


def init_model_bundle(force: bool = False) -> dict:
    global model_bundle, model_loaded
    with model_lock:
        if model_bundle is not None and not force:
            return model_bundle
        model_bundle = load_models()
        model_loaded = model_bundle is not None
        _sync_model_metrics(model_bundle)
        return model_bundle


def current_models() -> dict:
    with model_lock:
        if model_bundle is None:
            return init_model_bundle(force=True)
        return model_bundle


def build_output(msg: dict, models: dict | None = None) -> dict:
    bundle = models or current_models()
    feats = msg.get("payload", {})
    symbol = str(msg.get("symbol", "") or "")
    if not symbol or not isinstance(feats, dict):
        raise ValueError("malformed feature payload")

    vec = vectorize_features(feats)
    row = dict(feats)
    row["vec"] = vec

    anomaly_model = bundle["anomaly"]
    escalation_model = bundle["escalation"]
    raw_if, anomaly_norm, degraded_if = score_anomaly(getattr(anomaly_model, "model", None), row)
    esc_p, confidence, band, degraded_es = score_escalation(getattr(escalation_model, "model", None), row, anomaly_norm, degraded_if)
    comp = composite_risk(anomaly_norm, esc_p, row)
    pri = priority_score(comp, band, confidence, row)
    dq = float(row.get("dq_penalty", 0.0))
    action = recommended_action(pri, confidence, dq)
    safety = safety_level_from_bounds(pri, dq, confidence)
    explain = deterministic_explain(row)
    bucket_time = bucket_rfc3339(msg.get("event_time") or msg.get("ts"))
    produced_at = datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    feature_snapshot_hash = str(msg.get("feature_hash") or msg.get("feature_snapshot_hash") or snapshot_hash(feats))
    model_version = str(bundle.get("model_version") or _bundle_model_version(bundle))
    model_artifact_hash = str(bundle.get("model_artifact_hash") or _bundle_artifact_hash(bundle))
    scoring_run_id = str(uuid.uuid4())
    artifact_hash = stable_sha256([feature_snapshot_hash, model_artifact_hash, symbol, bucket_time])
    fallback_mode = degraded_if or degraded_es

    return {
        "symbol": symbol,
        "raw_anomaly_score": raw_if,
        "normalized_anomaly_score": anomaly_norm,
        "anomaly_score": anomaly_norm,
        "escalation_probability": esc_p,
        "escalation_prob": esc_p,
        "confidence": confidence,
        "expected_severity_band": band,
        "priority_score": pri,
        "recommended_action": action,
        "composite_risk": comp,
        "composite": comp,
        "safety_level": safety,
        "top_drivers": explain["top_drivers"],
        "explanation_text": explain["plain_language"],
        "feature_snapshot_hash": feature_snapshot_hash,
        "feature_set_version": str(msg.get("feature_set_version", feats.get("feature_set_version", "v2"))),
        "model_version": model_version,
        "artifact_hash": artifact_hash,
        "model_artifact_hash": model_artifact_hash,
        "scoring_run_id": scoring_run_id,
        "produced_at": produced_at,
        "calibration_version": str(bundle.get("calibration_version", "fallback")),
        "fallback_mode": fallback_mode,
        "dq_penalty": dq,
        "score": comp,
        "severity": band,
        "explanation": explain["plain_language"],
        "ts": bucket_time,
    }


@app.get("/")
def root():
    return jsonify({"service": "inference", "version": "dev", "links": ["/healthz", "/readyz", "/metrics", "/docs"]})


@app.get("/healthz")
def healthz():
    return "ok"


@app.get("/readyz")
def readyz():
    bundle = current_models()
    ready = kafka_ready and model_loaded
    return (
        jsonify(
            {
                "ready": ready,
                "fallback_mode": bool(bundle.get("fallback_mode", True)),
                "model_version": str(bundle.get("model_version", "")),
                "model_artifact_hash": str(bundle.get("model_artifact_hash", "")),
                "trained_at": str(bundle.get("trained_at", "")),
            }
        ),
        200 if ready else 503,
    )


@app.get("/metrics")
def metrics():
    return Response(generate_latest(), mimetype=CONTENT_TYPE_LATEST)


def run():
    global kafka_ready, running
    init_model_bundle(force=True)
    while running:
        try:
            consumer = KafkaConsumer(
                CONSUMER_TOPIC,
                bootstrap_servers=[BROKER],
                group_id=CONSUMER_GROUP,
                auto_offset_reset="earliest",
                value_deserializer=lambda v: json.loads(v.decode()),
                consumer_timeout_ms=-1,
            )
            producer = KafkaProducer(bootstrap_servers=[BROKER], value_serializer=lambda v: json.dumps(v).encode())
            kafka_ready = True
            logger.info("kafka consumer started; awaiting messages")
            msg_count = 0
            LOOP_RUNNING.set(1)
            try:
                for msg in consumer:
                    LOOP_RUNNING.set(1)
                    if not running:
                        logger.info("shutdown signal received")
                        break
                    try:
                        t0 = time.time()
                        out = build_output(msg.value, current_models())
                        SCORING_LATENCY.observe(time.time() - t0)
                        MESSAGES_CONSUMED.inc()
                        producer.send(PRODUCER_TOPIC, out)
                        producer.flush()
                        MESSAGES_PRODUCED.inc()
                        msg_count += 1
                        if msg_count % 50 == 0:
                            logger.info("processed %d messages", msg_count)
                    except Exception:
                        logger.exception("failed processing message %s", getattr(msg, "value", None))
                        PROCESSING_ERRORS.labels(status="error").inc()
            finally:
                LOOP_RUNNING.set(0)
                logger.info("consumer loop exiting; processed %d messages total", msg_count)
                consumer.close()
                producer.close()
        except Exception:
            kafka_ready = False
            LOOP_RUNNING.set(0)
            logger.exception("kafka consumer loop error")
            PROCESSING_ERRORS.labels(status="kafka_error").inc()
            time.sleep(2)


def signal_handler(sig, frame):
    del sig, frame
    global running
    running = False


init_model_bundle(force=True)


if __name__ == "__main__":
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    threading.Thread(target=run, daemon=True).start()
    app.run(host="0.0.0.0", port=int(os.getenv("PORT", "8090")), debug=False)
