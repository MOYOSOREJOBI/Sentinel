from src.scoring import priority_score


def test_priority_score_bounds():
    s=priority_score(1.0,"critical",0.0,{"anomaly_density_300":1.0,"incident_recurrence_900":1.0,"incident_age_s":99999})
    assert 0<=s<=100
