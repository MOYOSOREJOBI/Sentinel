from app import build_output


def test_bounds():
    feats={"z_ret_60":5.0,"volume_surprise_60":5.0,"ewma_vol_60":0.02,"dq_penalty":0.1,"volume_ratio_60":2.0,"anomaly_density_300":0.2,"realized_vol_300":0.05}
    out=build_output({"symbol":"AAPL","payload":feats})
    assert 0<=out["composite_risk"]<=1
    assert 0<=out["priority_score"]<=100
