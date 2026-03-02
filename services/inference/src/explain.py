from __future__ import annotations
try:
    import shap
except Exception:  # pragma: no cover
    shap = None


def build_tree_explainer(model):
    if shap is None:
        raise RuntimeError("shap unavailable")
    return shap.TreeExplainer(model, model_output="raw", feature_perturbation="tree_path_dependent")


def explain_tree_row(explainer, x_vec, feature_names, feature_baseline: dict):
    shap_vals = explainer.shap_values(x_vec)
    vals = shap_vals[1] if isinstance(shap_vals, list) else shap_vals
    row = vals[0]
    items = []
    for i, name in enumerate(feature_names):
        delta = float(x_vec[0][i]) - float(feature_baseline.get(name, 0.0))
        items.append({"feature": name, "contribution": float(row[i]), "delta_vs_baseline": delta})
    items.sort(key=lambda x: abs(x["contribution"]), reverse=True)
    top = items[:5]
    text = "Primary drivers: " + ", ".join(f'{it["feature"]} ({it["contribution"]:+.3f})' for it in top[:3])
    return {"top_drivers": top, "plain_language": text}


def deterministic_explain(feats: dict):
    items = [
        {"feature": "z_ret_60", "contribution": abs(float(feats.get("z_ret_60", 0.0))), "delta_vs_baseline": float(feats.get("z_ret_60", 0.0))},
        {"feature": "volume_surprise_60", "contribution": abs(float(feats.get("volume_surprise_60", 0.0))), "delta_vs_baseline": float(feats.get("volume_surprise_60", 0.0))},
        {"feature": "ewma_vol_60", "contribution": abs(float(feats.get("ewma_vol_60", 0.0))), "delta_vs_baseline": float(feats.get("ewma_vol_60", 0.0))},
        {"feature": "dq_penalty", "contribution": -abs(float(feats.get("dq_penalty", 0.0))), "delta_vs_baseline": float(feats.get("dq_penalty", 0.0))},
    ]
    items.sort(key=lambda x: abs(x["contribution"]), reverse=True)
    return {"top_drivers": items[:5], "plain_language": "Primary drivers: z_ret_60, volume_surprise_60, ewma_vol_60"}
