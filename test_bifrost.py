"""Thin pytest wrapper around the Bifrost standalone runner.

`python3 bifrost.py` is the source of truth; this just lets `pytest` drive the
same corpus so it slots into CI / IDE test runners. Each case becomes one
parametrised test. A KNOWN DIVERGENCE is asserted to be exactly that (DIVERGE),
never silently treated as a pass-or-fail. Heavy cases are skipped unless
BIFROST_HEAVY is set in the environment.

Run:  pytest            (light cases)
      BIFROST_HEAVY=1 pytest -k ratatoskr   (include the heavy parity case)
"""

import os
import pytest

import bifrost


_HEAVY = bool(os.environ.get("BIFROST_HEAVY"))

_adapters = bifrost.load_adapters()
_available, _skipped = bifrost.discover(_adapters)
_cases = bifrost.load_cases(include_heavy=_HEAVY)


def _run(case):
    if case["mode"] == "special-hush":
        return bifrost.run_hush_divergence(case, _available)
    if case["expect"] == "ratatoskr-parity":
        return bifrost.run_ratatoskr_parity(case, _available)
    return bifrost.evaluate_case(case, _available, bifrost.TIMEOUT_HEAVY)


@pytest.mark.skipif(not _available, reason="no Shen implementations available")
@pytest.mark.parametrize("case", _cases, ids=[c["name"] for c in _cases])
def test_case(case):
    if case.get("heavy") and not _HEAVY:
        pytest.skip("heavy case; set BIFROST_HEAVY=1 to run")
    res = _run(case)
    verdict = res["verdict"]
    if verdict == "FAIL":
        pytest.fail("%s: %s" % (case["name"], res["detail"]))
    elif verdict == "DIVERGE":
        # A documented cross-impl difference. Surface it (xfail-style) but do
        # not fail the suite -- this is the point of the divergence corpus.
        pytest.xfail("KNOWN DIVERGENCE -- %s: %s" % (case["name"], res["detail"]))
    # else PASS


def test_at_least_one_impl_available():
    assert _available, "Bifrost found no Shen implementations to drive: %r" % _skipped
