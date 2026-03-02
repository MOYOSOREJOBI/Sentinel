from __future__ import annotations

FEATURE_COLUMNS = tuple(
    sorted(
        [
            "anomaly_density_300",
            "dq_penalty",
            "duplicate_ratio",
            "ewma_var_300",
            "ewma_var_60",
            "ewma_vol_300",
            "ewma_vol_60",
            "late_ratio",
            "log_ret_1m",
            "log_ret_1s",
            "log_ret_5m",
            "mean_ret_300",
            "mean_ret_60",
            "mean_vol_60",
            "missingness_ratio",
            "out_of_order_ratio",
            "realized_var_300",
            "realized_vol_300",
            "ret_1m",
            "ret_1s",
            "ret_5m",
            "staleness_score",
            "std_ret_300",
            "std_ret_60",
            "std_vol_60",
            "tick_gap_s",
            "volume_ratio_60",
            "volume_surprise_60",
            "z_ret_300",
            "z_ret_60",
        ]
    )
)


def vectorize_features(feats: dict) -> list[float]:
    return [float(feats.get(name, 0.0) or 0.0) for name in FEATURE_COLUMNS]
