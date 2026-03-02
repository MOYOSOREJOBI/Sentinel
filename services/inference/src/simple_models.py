from __future__ import annotations

import math


def _clip(value: float, lo: float, hi: float) -> float:
    return max(lo, min(hi, value))


def _sigmoid(value: float) -> float:
    if value >= 0:
        z = math.exp(-value)
        return 1.0 / (1.0 + z)
    z = math.exp(value)
    return z / (1.0 + z)


class RobustAnomalyModel:
    def __init__(self, medians: list[float], scales: list[float], score_shift: float = 0.10, score_span: float = 0.60):
        self.medians = [float(v) for v in medians]
        self.scales = [max(float(v), 1e-6) for v in scales]
        self.score_shift = float(score_shift)
        self.score_span = float(score_span)

    def decision_function(self, rows: list[list[float]]) -> list[float]:
        out: list[float] = []
        for row in rows:
            robust_parts = []
            for i, center in enumerate(self.medians):
                value = float(row[i]) if i < len(row) else 0.0
                robust_z = abs(value-center) / self.scales[i]
                robust_parts.append(_clip(robust_z/6.0, 0.0, 1.0))
            anomaly_norm = sum(robust_parts)/max(len(robust_parts), 1)
            raw_if = self.score_shift - (self.score_span * anomaly_norm)
            out.append(raw_if)
        return out


class CalibratedLogisticModel:
    def __init__(
        self,
        means: list[float],
        scales: list[float],
        weights: list[float],
        bias: float,
        calibration_a: float = 1.0,
        calibration_b: float = 0.0,
    ):
        self.means = [float(v) for v in means]
        self.scales = [max(float(v), 1e-6) for v in scales]
        self.weights = [float(v) for v in weights]
        self.bias = float(bias)
        self.calibration_a = float(calibration_a)
        self.calibration_b = float(calibration_b)

    def _logit(self, row: list[float]) -> float:
        total = self.bias
        for i, weight in enumerate(self.weights):
            value = float(row[i]) if i < len(row) else 0.0
            total += ((value-self.means[i]) / self.scales[i]) * weight
        return total

    def predict_proba(self, rows: list[list[float]]) -> list[list[float]]:
        out: list[list[float]] = []
        for row in rows:
            raw_logit = self._logit(row)
            calibrated_logit = (self.calibration_a * raw_logit) + self.calibration_b
            p = _clip(_sigmoid(calibrated_logit), 1e-6, 1.0-1e-6)
            out.append([1.0-p, p])
        return out
