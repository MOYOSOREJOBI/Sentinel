import pytest

from app import build_output


def test_inference_output_shape():
    out = build_output({'symbol': 'AAPL', 'payload': {'z_ret_60': 0.1, 'volume_surprise_60': 0.2, 'ewma_vol_60': 0.01, 'dq_penalty': 0.0, 'volume_ratio_60': 1.0, 'anomaly_density_300': 0.1, 'realized_vol_300': 0.02}})
    assert 'symbol' in out and 'score' in out and 'severity' in out and 'explanation' in out


def test_malformed_input_fails_safely():
    with pytest.raises(ValueError):
        build_output({'payload': {}})
