import pytest

from evals.harness import run_evaluations


@pytest.mark.integration
def test_eval_harness_passes() -> None:
    summary = run_evaluations()
    assert summary["passed"] is True
    assert summary["passed_cases"] == summary["total_cases"]
    assert summary["total_cases"] >= 4
