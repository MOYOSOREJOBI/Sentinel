from app import build_output


class A:
    def decision_function(self, arr):
        return [-0.2]


class E:
    def predict_proba(self, arr):
        return [[0.1,0.9]]


def test_model_artifacts_change_runtime_predictions():
    feats = {"z_ret_60":1.0,"volume_surprise_60":1.0,"ewma_vol_60":0.01,"dq_penalty":0.0,"volume_ratio_60":1.0,"anomaly_density_300":0.1,"realized_vol_300":0.03}
    base = build_output({"symbol":"AAPL","payload":feats}, models={"anomaly":type("M",(),{"model":None,"version":"f","artifact_hash":"a"})(),"escalation":type("M",(),{"model":None,"version":"f","artifact_hash":"b"})()})
    mod = build_output({"symbol":"AAPL","payload":feats}, models={"anomaly":type("M",(),{"model":A(),"version":"m","artifact_hash":"x"})(),"escalation":type("M",(),{"model":E(),"version":"m","artifact_hash":"y"})()})
    assert base["priority_score"] != mod["priority_score"]
