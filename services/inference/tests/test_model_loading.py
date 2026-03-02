from src.models import load_pickle_model


def test_default_model_loaded_without_file():
    model = load_pickle_model('/does/not/exist.pkl', 'fallback-v1')
    assert model.degraded is True
    assert model.model is None
