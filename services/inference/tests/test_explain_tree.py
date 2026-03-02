from src.explain import deterministic_explain


def test_explain_fallback_has_drivers():
    r=deterministic_explain({"z_ret_60":2})
    assert len(r["top_drivers"])>=3
