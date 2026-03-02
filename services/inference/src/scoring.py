import math


def clip(x: float, lo: float, hi: float) -> float:
    return max(lo, min(hi, x))


def score_anomaly(if_model, x_row: dict) -> tuple[float, float, bool]:
    if if_model is None:
        z = abs(float(x_row["z_ret_60"]))
        vol_s = abs(float(x_row["volume_surprise_60"]))
        ewma = float(x_row["ewma_vol_60"])
        dq = float(x_row["dq_penalty"])
        proxy = 0.50 * clip(z / 6.0, 0, 1) + 0.30 * clip(abs(vol_s) / 6.0, 0, 1) + 0.20 * clip(ewma / 0.04, 0, 1)
        proxy = clip(proxy - 0.35 * dq, 0, 1)
        raw_if = 0.10 - 0.60 * proxy
        return raw_if, proxy, True

    raw_if = float(if_model.decision_function([x_row["vec"]])[0])
    anomaly_norm = clip((-raw_if + 0.10) / 0.60, 0.0, 1.0)
    return raw_if, anomaly_norm, False


def score_escalation(gbm_model, feats: dict, anomaly_norm: float, degraded_if: bool) -> tuple[float, float, str, bool]:
    dq = float(feats["dq_penalty"])
    z = clip(abs(float(feats["z_ret_60"])) / 6.0, 0, 1)
    ewma = clip(float(feats["ewma_vol_60"]) / 0.04, 0, 1)
    vol_ratio = clip((float(feats["volume_ratio_60"]) - 1.0) / 4.0, 0, 1)
    pressure = clip(float(feats["anomaly_density_300"]), 0, 1)

    if gbm_model is None:
        logit = -1.20 + 1.10 * anomaly_norm + 0.70 * z + 0.55 * ewma + 0.40 * vol_ratio + 0.35 * pressure - 0.90 * dq
        p = 1.0 / (1.0 + math.exp(-logit))
        degraded = True
    else:
        proba = gbm_model.predict_proba([feats["vec"]])[0]
        p = float(proba[1])
        degraded = False

    model_degraded = 1.0 if degraded or degraded_if else 0.0
    confidence = 1.0 - clip(0.55 * dq + 0.25 * float(feats.get("missing_features_ratio", 0.0)) + 0.20 * model_degraded, 0.0, 0.85)

    if p >= 0.85:
        band = "critical"
    elif p >= 0.65:
        band = "high"
    elif p >= 0.40:
        band = "elevated"
    else:
        band = "stable"

    return p, confidence, band, degraded


def compute_vol_ctx(feats: dict) -> float:
    short_vol = clip(float(feats["ewma_vol_60"]) / 0.04, 0, 1)
    med_vol = clip(float(feats["realized_vol_300"]) / 0.08, 0, 1)
    return clip(0.60 * short_vol + 0.40 * med_vol, 0, 1)


def compute_pressure(feats: dict) -> float:
    ad = clip(float(feats["anomaly_density_300"]), 0, 1)
    recur = clip(float(feats.get("incident_recurrence_900", 0.0)), 0, 1)
    return clip(0.65 * ad + 0.35 * recur, 0, 1)


def compute_business_weight(feats: dict) -> float:
    watch = 1.0 if feats.get("on_watchlist") else 0.0
    priority_country = 1.0 if feats.get("country_code") in {"US", "GB", "DE", "JP"} else 0.0
    tier1_sector = 1.0 if feats.get("sector") in {"Finance", "Energy", "Technology"} else 0.0
    return clip(0.40 * watch + 0.35 * priority_country + 0.25 * tier1_sector, 0, 1)


def composite_risk(anomaly_norm: float, escalation_p: float, feats: dict) -> float:
    dq = float(feats["dq_penalty"])
    vol_ctx = compute_vol_ctx(feats)
    pressure = compute_pressure(feats)
    business = compute_business_weight(feats)
    return clip(0.35 * anomaly_norm + 0.30 * escalation_p + 0.15 * vol_ctx + 0.10 * pressure + 0.10 * business - 0.20 * dq, 0.0, 1.0)


SEVERITY_INDEX = {"stable": 0.15, "elevated": 0.45, "high": 0.75, "critical": 1.00, "data_unreliable": 0.20}


def sla_pressure(age_seconds: float, sla_seconds: float = 900.0) -> float:
    return clip(age_seconds / max(sla_seconds, 1.0), 0.0, 1.0)


def priority_score(composite: float, severity_band: str, confidence: float, feats: dict) -> float:
    sev = SEVERITY_INDEX.get(severity_band, 0.15)
    pressure = compute_pressure(feats)
    age_p = sla_pressure(float(feats.get("incident_age_s", 0.0)))
    score = 100.0 * (0.46 * composite + 0.18 * sev + 0.14 * age_p + 0.12 * pressure + 0.10 * (1.0 - confidence))
    return clip(score, 0.0, 100.0)


def recommended_action(priority: float, confidence: float, dq_penalty: float) -> str:
    if dq_penalty >= 0.60 or confidence < 0.35:
        return "check_data"
    if priority >= 85:
        return "promote_to_case"
    if priority >= 70:
        return "review_now"
    if priority >= 50:
        return "watch"
    return "suppress_candidate"
