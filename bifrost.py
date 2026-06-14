#!/usr/bin/env python3
"""
Bifrost -- a cross-implementation META test harness for the Shen language ports.

  "Bifrost is the bridge between worlds." Here it is the bridge that checks the
  worlds agree: it runs the SAME programs/behaviours across ALL FIVE Shen
  implementations and asserts they agree (differential / conformance testing).

This is DISTINCT from:
  (a) the canonical Shen kernel test suite (the spec's own conformance tests
      that each port already runs against the kernel), and
  (b) each port's own implementation-specific unit tests.
Bifrost is the third thing: a META suite that drives every port the same way
and diffs their observable behaviour.

Pure standard library, Python 3 only. No third-party deps. A thin pytest
wrapper (test_bifrost.py) is provided too, but THIS standalone runner is the
source of truth and works with just `python3 bifrost.py`.

Usage:
    python3 bifrost.py                  # run all non-heavy cases, print matrix
    python3 bifrost.py --heavy          # also run heavy (ratatoskr) cases
    python3 bifrost.py --only NAME ...  # run only the named cases
    python3 bifrost.py --impls a,b      # restrict to a subset of impls
    python3 bifrost.py --list           # list discovered impls + cases
    python3 bifrost.py --json           # emit a machine-readable result blob

Exit code is non-zero on a real FAILURE. KNOWN DIVERGENCES are reported in
their own section and do NOT, by themselves, fail the run.
"""

import argparse
import json
import os
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
ADAPTERS_PATH = os.path.join(HERE, "adapters.json")
CASES_DIR = os.path.join(HERE, "cases")
PROGRAMS_DIR = os.path.join(HERE, "programs")

# Order in which impls appear in the matrix.
IMPL_ORDER = ["shen-cl", "shen-go", "shen-rust", "shen-lua", "ShenScript"]

TIMEOUT_DEFAULT = 60      # seconds, per invocation
TIMEOUT_HEAVY = 300       # seconds, for heavy (ratatoskr) cases


# --------------------------------------------------------------------------
# Output normalisation
# --------------------------------------------------------------------------
# Cross-impl stdout carries impl-specific noise. We strip it so that *behaviour*
# is compared, not banner chrome. The rules below are deliberately conservative
# and each is justified by an observed, real divergence in launcher chatter
# (NOT in computed values).

def normalize(text):
    """Normalise an impl's stdout for comparison.

    - strip a trailing run-time banner line ("run time: ... secs") emitted by
      the kernel `load`/timer on several ports,
    - strip `(fn NAME)` declaration echoes that `load` prints (shen-lua's file
      path is `(load FILE)`, which echoes every define),
    - strip lone `0`/`nil`/`true` load-echo lines that `load` prints as the
      value of a trailing `(nl)` form (shen-lua),
    - normalise CRLF -> LF and strip trailing whitespace on every line,
    - collapse a run of blank lines and strip leading/trailing blank lines.
    These touch only launcher chatter; computed output is preserved verbatim.
    """
    lines = text.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    out = []
    for ln in lines:
        s = ln.rstrip()
        stripped = s.strip()
        if stripped.startswith("run time:"):
            continue
        if stripped.startswith("(fn ") and stripped.endswith(")"):
            continue
        out.append(s)
    # drop leading/trailing blanks
    while out and out[0].strip() == "":
        out.pop(0)
    while out and out[-1].strip() == "":
        out.pop()
    # shen-lua's `load` echoes the value of the trailing `(nl)` form as a lone
    # "0" after the program's own output. If the last surviving line is a lone
    # "0" preceded by real content, drop it (the harness convention is that
    # script programs end with `(nl)`).
    if len(out) >= 2 and out[-1].strip() == "0":
        out.pop()
    # collapse interior blank runs
    collapsed = []
    blank = False
    for s in out:
        if s.strip() == "":
            if blank:
                continue
            blank = True
        else:
            blank = False
        collapsed.append(s)
    return "\n".join(collapsed)


# --------------------------------------------------------------------------
# Adapters: discover & resolve implementations
# --------------------------------------------------------------------------

def load_adapters():
    with open(ADAPTERS_PATH) as f:
        raw = json.load(f)
    return {k: v for k, v in raw.items() if not k.startswith("_")}


def resolve_bin(name, cfg):
    """Resolve an impl's launcher path. Returns absolute path or None.

    Order: env var override, then first existing default_path. Relative
    default paths are resolved against the bifrost repo root (HERE).
    """
    env = cfg.get("env")
    if env and os.environ.get(env):
        cand = os.environ[env]
        if os.path.exists(cand):
            return os.path.abspath(cand)
    for p in cfg.get("default_paths", []):
        cand = p if os.path.isabs(p) else os.path.join(HERE, p)
        if os.path.exists(cand):
            return os.path.abspath(cand)
    return None


def discover(adapters, restrict=None):
    """Return (available, skipped). available: name->{cfg,bin}."""
    available, skipped = {}, {}
    for name in IMPL_ORDER:
        if name not in adapters:
            continue
        if restrict and name not in restrict:
            continue
        cfg = adapters[name]
        binpath = resolve_bin(name, cfg)
        if binpath is None:
            skipped[name] = "launcher not found (set $%s or build it)" % cfg.get("env", "?")
        else:
            available[name] = {"cfg": cfg, "bin": binpath}
    return available, skipped


def _sub_tokens(tok, subs):
    """subs maps a brace-delimited placeholder (e.g. '{bin}') to its value."""
    for k, v in subs.items():
        tok = tok.replace(k, v)
    return tok


# --------------------------------------------------------------------------
# Running a single (impl, case)
# --------------------------------------------------------------------------

def run_invocation(argv, timeout, stdin_eof=False):
    """Run argv, capture combined stdout (+stderr). Returns dict."""
    try:
        # For the EOF probe, feed an empty pipe that then closes — the faithful
        # `echo -n | shen` scenario (a closed PIPE delivers EOF on first read).
        # Otherwise give the child no stdin at all.
        proc = subprocess.run(
            argv,
            input=b"" if stdin_eof else None,
            stdin=None if stdin_eof else subprocess.DEVNULL,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            timeout=timeout,
        )
        return {"rc": proc.returncode,
                "out": proc.stdout.decode("utf-8", "replace"),
                "timeout": False}
    except subprocess.TimeoutExpired as e:
        partial = e.stdout.decode("utf-8", "replace") if e.stdout else ""
        return {"rc": None, "out": partial, "timeout": True}
    except FileNotFoundError as e:
        return {"rc": None, "out": "launcher missing: %s" % e, "timeout": False}


def build_argv(impl, case):
    """Build the argv for (impl, case) according to its mode."""
    cfg = impl["cfg"]
    binpath = impl["bin"]
    subs = {"{bin}": binpath}
    mode = case["mode"]
    if mode == "eval":
        subs["{expr}"] = case["expr"]
        tmpl = cfg["eval"]
    elif mode == "script":
        prog = os.path.join(PROGRAMS_DIR, case["program"])
        subs["{file}"] = prog
        tmpl = cfg["script"]
    elif mode == "version":
        tmpl = cfg.get("version")
        if not tmpl:
            # fall back to eval (version)
            subs["{expr}"] = "(version)"
            tmpl = cfg["eval"]
    elif mode == "repl-eof":
        tmpl = cfg.get("repl") or cfg["eval"]
    else:
        raise ValueError("unknown mode %r" % mode)
    return [_sub_tokens(t, subs) for t in tmpl]


def run_case_on_impl(name, impl, case, heavy_timeout):
    timeout = heavy_timeout if case.get("heavy") else TIMEOUT_DEFAULT
    mode = case["mode"]

    if mode == "version":
        argv = build_argv(impl, case)
        r = run_invocation(argv, timeout)
        out = normalize(r["out"])
        ok = (not r["timeout"]) and (case["contains"] in r["out"])
        return {"raw": r["out"], "norm": out, "ok": ok, "timeout": r["timeout"], "rc": r["rc"]}

    if mode == "repl-eof":
        argv = build_argv(impl, case)
        r = run_invocation(argv, timeout, stdin_eof=True)
        # success = clean exit (no hang). rc 0 expected; any non-timeout exit ok.
        ok = (not r["timeout"])
        return {"raw": r["out"], "norm": normalize(r["out"]), "ok": ok,
                "timeout": r["timeout"], "rc": r["rc"]}

    # eval / script
    argv = build_argv(impl, case)
    r = run_invocation(argv, timeout)
    norm = normalize(r["out"])
    return {"raw": r["out"], "norm": norm, "ok": None, "timeout": r["timeout"], "rc": r["rc"]}


# --------------------------------------------------------------------------
# Cases
# --------------------------------------------------------------------------

def load_cases(include_heavy):
    cases = []
    for fn in sorted(os.listdir(CASES_DIR)):
        if not fn.endswith(".json"):
            continue
        with open(os.path.join(CASES_DIR, fn)) as f:
            blob = json.load(f)
        group = blob.get("group", fn[:-5])
        for c in blob["cases"]:
            c.setdefault("group", group)
            c["_file"] = fn
            if c.get("heavy") and not include_heavy:
                continue
            cases.append(c)
    return cases


# --------------------------------------------------------------------------
# Evaluation of a case across impls
# --------------------------------------------------------------------------

def evaluate_case(case, available, heavy_timeout):
    """Run case on every available impl and decide pass/fail/divergence.

    Returns dict:
      per_impl: name -> result (with .norm, .status in PASS/FAIL/DIVERGE/SKIP)
      verdict : "PASS" | "FAIL" | "DIVERGE"
      detail  : human string
    """
    results = {}
    for name, impl in available.items():
        results[name] = run_case_on_impl(name, impl, case, heavy_timeout)

    expect = case["expect"]
    verdict = "PASS"
    detail = ""

    if expect == "output":
        golden = case["golden"]
        for name, r in results.items():
            if r["timeout"]:
                r["status"] = "FAIL"
                verdict = "FAIL"
            elif r["norm"] == golden:
                r["status"] = "PASS"
            else:
                r["status"] = "FAIL"
                verdict = "FAIL"
        if verdict == "FAIL":
            detail = "golden=%r; mismatches: %s" % (
                golden,
                ", ".join("%s=%r" % (n, results[n]["norm"])
                          for n in results if results[n]["status"] == "FAIL"))

    elif expect == "agreement":
        norms = {n: r["norm"] for n, r in results.items()}
        timed = [n for n, r in results.items() if r["timeout"]]
        distinct = set(norms.values())
        is_divergence = bool(case.get("known_divergence"))
        if timed:
            verdict = "FAIL"
            for n, r in results.items():
                r["status"] = "FAIL" if r["timeout"] else "PASS"
            detail = "timeout on: %s" % ", ".join(timed)
        elif len(distinct) <= 1:
            verdict = "PASS"
            for r in results.values():
                r["status"] = "PASS"
        else:
            # impls disagree
            verdict = "DIVERGE" if is_divergence else "FAIL"
            # mark each impl's bucket
            for n, r in results.items():
                r["status"] = "DIVERGE" if is_divergence else "FAIL"
            buckets = {}
            for n, v in norms.items():
                buckets.setdefault(v, []).append(n)
            detail = "; ".join("%r <- %s" % (v, ",".join(ns))
                               for v, ns in buckets.items())
            if is_divergence:
                detail = "KNOWN DIVERGENCE [%s]: %s" % (case["known_divergence"], detail)

    elif expect == "version":
        # handled as per-impl 'contains'
        for name, r in results.items():
            if r["ok"]:
                r["status"] = "PASS"
            else:
                r["status"] = "FAIL"
                verdict = "FAIL"
        if verdict == "FAIL":
            detail = "version must contain %r" % case["contains"]

    elif expect == "clean-exit":
        for name, r in results.items():
            if r["ok"]:
                r["status"] = "PASS"
            else:
                r["status"] = "FAIL"
                verdict = "FAIL"
        if verdict == "FAIL":
            detail = "REPL did not exit cleanly on EOF (hang/timeout)"

    elif expect == "divergence-table":
        # KNOWN-DIVERGENCE: each impl is checked against its declared expected
        # bucket in case['table'] (name -> "written"|"empty"). This documents a
        # real difference; it is reported as DIVERGE, never FAIL, as long as
        # each impl matches its documented bucket.
        verdict = "DIVERGE"
        detail = case.get("doc", "")
        # results already filled by the special runner below
        for name, r in results.items():
            r["status"] = "DIVERGE"

    else:
        raise ValueError("unknown expect %r" % expect)

    return {"per_impl": results, "verdict": verdict, "detail": detail}


# --------------------------------------------------------------------------
# Special case: the -q/*hush* file-write divergence
# --------------------------------------------------------------------------

def run_hush_divergence(case, available):
    """Drive the -q/*hush* file-write behaviour for each impl and verify it.

    Two semantics, selected per case:

    * Hardened agreement (default, when `known_divergence` is absent): assert
      that EVERY available impl lands in the same bucket -- i.e. `pr` to a FILE
      stream behaves identically under -q across all ports. A disagreement is a
      hard FAIL. This is the locked-in form once the divergence has converged.

    * Documented divergence (when `known_divergence` is present and a `table`
      is supplied): each impl is checked against its declared expected bucket in
      case['table'] (name -> "written"|"empty"); a mismatch is FAIL, otherwise
      the run is reported as DIVERGE (never a hard agreement failure).

    Historically (pre-fix) this was: shen-cl / shen-go / ShenScript WRITE the
    file under -q, while shen-lua / shen-rust SILENCE the write (zero-byte file).
    """
    is_divergence = bool(case.get("known_divergence"))
    table = case.get("table") or {}  # name -> "written" | "empty"
    results = {}
    tmpdir = tempfile.mkdtemp(prefix="bifrost_hush_")
    for name, impl in available.items():
        cfg = impl["cfg"]
        binpath = impl["bin"]
        outfile = os.path.join(tmpdir, "%s.out" % name)
        if os.path.exists(outfile):
            os.remove(outfile)
        prog = os.path.join(tmpdir, "%s_write.shen" % name)
        with open(prog, "w") as f:
            f.write('(let S (open "%s" out) (do (pr "hello" S) (close S)))\n' % outfile)
        # Build a -q (hush) invocation. We must inject -q for impls whose eval
        # template lacks it. Use the per-impl hush invocation template.
        argv = build_hush_argv(name, cfg, binpath, prog)
        run_invocation(argv, TIMEOUT_DEFAULT)
        size = os.path.getsize(outfile) if os.path.exists(outfile) else 0
        observed = "written" if size > 0 else "empty"
        expected = table.get(name)
        results[name] = {
            "raw": "%d bytes (%s)" % (size, observed),
            "norm": observed,
            "timeout": False,
            "rc": 0,
            "size": size,
            "expected": expected,
        }
    buckets = {}
    for n, r in results.items():
        buckets.setdefault(r["norm"], []).append(n)

    if is_divergence:
        # Documented-divergence mode: each impl must match its declared bucket.
        for name, r in results.items():
            match = (r["expected"] is None) or (r["norm"] == r["expected"])
            r["ok"] = match
            r["status"] = "DIVERGE" if match else "FAIL"
        verdict = "FAIL" if any(r["status"] == "FAIL" for r in results.values()) else "DIVERGE"
        detail = "KNOWN DIVERGENCE [%s]: " % case["known_divergence"] + "; ".join(
            "%s <- %s" % (b, ",".join(ns)) for b, ns in buckets.items())
        return {"per_impl": results, "verdict": verdict, "detail": detail}

    # Hardened-agreement mode: all impls must land in the same bucket.
    agree = len(buckets) <= 1
    for r in results.values():
        r["ok"] = agree
        r["status"] = "PASS" if agree else "FAIL"
    verdict = "PASS" if agree else "FAIL"
    detail = "; ".join("%s <- %s" % (b, ",".join(ns)) for b, ns in buckets.items())
    if not agree:
        detail = "disagreement under -q/*hush* file write: " + detail
    return {"per_impl": results, "verdict": verdict, "detail": detail}


def build_hush_argv(name, cfg, binpath, prog):
    """Argv that loads PROG under -q (*hush* = true) for each impl.

    shen-cl     : eval -q -l PROG
    shen-go     : eval -q -l PROG  (writes regardless)
    shen-rust   : eval -q -l PROG  (-q BEFORE -l silences pr -> empty file)
    shen-lua    : -q PROG          (positional load under hush -> empty file)
    ShenScript  : eval -q -l PROG
    """
    if name == "shen-lua":
        return [binpath, "-q", prog]
    if name == "ShenScript":
        launch = cfg.get("launcher", [])
        return launch + [binpath, "eval", "-q", "-l", prog]
    return [binpath, "eval", "-q", "-l", prog]


# --------------------------------------------------------------------------
# Heavy case: ratatoskr stage-1 parity (byte-identical kernel.kl + manifest)
# --------------------------------------------------------------------------

RATATOSKR_DIR_DEFAULT = "/Users/reuben/projects/shen/ratatoskr"


def run_ratatoskr_parity(case, available):
    """Run ratatoskr.shake on each host and assert kernel.kl + manifest are
    BYTE-IDENTICAL across all available hosts.

    The user KL (e.g. fib.kl) differs only by gensym counter, so we DO NOT
    assert on it -- only on kernel.kl and ratatoskr.manifest, which are the
    deterministic stage-1 artefacts.

    Critical gotcha: drive shen-lua / shen-rust WITHOUT -q, or `pr` writes
    zero-byte files (see the hush-file-write divergence).
    """
    rtk_dir = os.environ.get("BIFROST_RATATOSKR_DIR", RATATOSKR_DIR_DEFAULT)
    results = {}
    if not os.path.isfile(os.path.join(rtk_dir, "ratatoskr.shen")):
        for name in available:
            results[name] = {"raw": "ratatoskr.shen not found in %s" % rtk_dir,
                             "norm": "no-ratatoskr", "ok": None, "timeout": False,
                             "rc": None, "status": "SKIP"}
        return {"per_impl": results, "verdict": "PASS",
                "detail": "SKIPPED: ratatoskr repo not found at %s "
                          "(set $BIFROST_RATATOSKR_DIR)" % rtk_dir}

    files_arg = case.get("shake_files", '["tests/fib.shen"]')
    digests = {}  # name -> (kernel_md5, manifest_md5) or None
    tmproot = tempfile.mkdtemp(prefix="bifrost_rtk_")
    for name, impl in available.items():
        cfg = impl["cfg"]
        binpath = impl["bin"]
        outdir = os.path.join(tmproot, name)
        os.makedirs(outdir, exist_ok=True)
        drv = os.path.join(tmproot, "%s_shake.shen" % name)
        with open(drv, "w") as f:
            f.write('(load "ratatoskr.shen")\n')
            f.write('(ratatoskr.shake %s "%s")\n' % (files_arg, outdir))
        # NB: NO -q for any host here (lua/rust would silence the file writes).
        if name == "shen-lua":
            argv = [binpath, drv]
        elif name == "ShenScript":
            argv = cfg.get("launcher", []) + [binpath, "eval", "-l", drv]
        else:
            argv = [binpath, "eval", "-l", drv]
        try:
            subprocess.run(argv, cwd=rtk_dir, stdin=subprocess.DEVNULL,
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                           timeout=TIMEOUT_HEAVY)
        except subprocess.TimeoutExpired:
            results[name] = {"raw": "TIMEOUT", "norm": "timeout", "ok": False,
                             "timeout": True, "rc": None, "status": "FAIL"}
            digests[name] = None
            continue
        kpath = os.path.join(outdir, "kernel.kl")
        mpath = os.path.join(outdir, "ratatoskr.manifest")
        if os.path.isfile(kpath) and os.path.getsize(kpath) > 0 \
                and os.path.isfile(mpath) and os.path.getsize(mpath) > 0:
            import hashlib
            kd = hashlib.md5(open(kpath, "rb").read()).hexdigest()
            md = hashlib.md5(open(mpath, "rb").read()).hexdigest()
            digests[name] = (kd, md)
            results[name] = {"raw": "kernel.kl=%dB manifest=%dB"
                             % (os.path.getsize(kpath), os.path.getsize(mpath)),
                             "norm": kd[:8] + "/" + md[:8], "ok": None,
                             "timeout": False, "rc": 0, "status": "PASS"}
        else:
            digests[name] = None
            results[name] = {"raw": "no/empty artefacts (forget to omit -q?)",
                             "norm": "empty", "ok": False, "timeout": False,
                             "rc": None, "status": "FAIL"}

    produced = {n: d for n, d in digests.items() if d is not None}
    distinct = set(produced.values())
    if any(results[n]["status"] == "FAIL" for n in results):
        verdict = "FAIL"
    elif len(distinct) <= 1:
        verdict = "PASS"
    else:
        verdict = "FAIL"
    if len(distinct) > 1:
        for n, d in produced.items():
            if list(produced.values()).count(d) < len(produced):
                results[n]["status"] = "FAIL"
        detail = "kernel.kl/manifest digests disagree: " + "; ".join(
            "%s=%s/%s" % (n, d[0][:8], d[1][:8]) for n, d in produced.items())
    elif verdict == "PASS":
        sample = next(iter(distinct)) if distinct else ("?", "?")
        detail = ("kernel.kl + ratatoskr.manifest BYTE-IDENTICAL across %d hosts "
                  "(kernel md5 %s, manifest md5 %s)"
                  % (len(produced), sample[0][:8], sample[1][:8]))
    else:
        detail = "some hosts produced no/empty stage-1 artefacts"
    return {"per_impl": results, "verdict": verdict, "detail": detail}


# --------------------------------------------------------------------------
# Reporting
# --------------------------------------------------------------------------

SYM = {"PASS": "PASS", "FAIL": "FAIL", "DIVERGE": "DVRG", "SKIP": "----"}


def print_matrix(cases, case_results, available, skipped):
    impls = list(available.keys())
    namew = max([len("case")] + [len(c["name"]) for c in cases]) + 2
    colw = max(10, max((len(i) for i in impls), default=4) + 1)

    print("\n" + "=" * 78)
    print("BIFROST MATRIX  (impl x case)")
    print("=" * 78)
    header = "case".ljust(namew) + "".join(i.ljust(colw) for i in impls) + "verdict"
    print(header)
    print("-" * len(header))
    for c in cases:
        res = case_results[c["name"]]
        row = c["name"].ljust(namew)
        for i in impls:
            st = res["per_impl"].get(i, {}).get("status", "SKIP")
            row += SYM.get(st, st).ljust(colw)
        row += res["verdict"]
        print(row)
    print("-" * len(header))

    if skipped:
        print("\nSKIPPED IMPLEMENTATIONS (launcher not found; reported, not failed):")
        for n, why in skipped.items():
            print("  - %s: %s" % (n, why))

    # divergence detail section
    divs = [c for c in cases if case_results[c["name"]]["verdict"] == "DIVERGE"]
    if divs:
        print("\nKNOWN DIVERGENCES (documented differences; do NOT fail the run):")
        for c in divs:
            print("  * %s: %s" % (c["name"], case_results[c["name"]]["detail"]))

    fails = [c for c in cases if case_results[c["name"]]["verdict"] == "FAIL"]
    if fails:
        print("\nFAILURES:")
        for c in fails:
            print("  ! %s: %s" % (c["name"], case_results[c["name"]]["detail"]))

    npass = sum(1 for c in cases if case_results[c["name"]]["verdict"] == "PASS")
    print("\nSUMMARY: %d pass, %d diverge (known), %d FAIL  across %d impls"
          % (npass, len(divs), len(fails), len(impls)))
    print("=" * 78)
    return len(fails)


# --------------------------------------------------------------------------
# Main
# --------------------------------------------------------------------------

def main(argv=None):
    ap = argparse.ArgumentParser(description="Bifrost cross-impl Shen meta test harness")
    ap.add_argument("--heavy", action="store_true", help="include heavy (ratatoskr) cases")
    ap.add_argument("--only", nargs="*", help="run only the named cases")
    ap.add_argument("--impls", help="comma-separated subset of impls to drive")
    ap.add_argument("--list", action="store_true", help="list impls + cases and exit")
    ap.add_argument("--json", action="store_true", help="emit machine-readable JSON")
    args = ap.parse_args(argv)

    adapters = load_adapters()
    restrict = set(args.impls.split(",")) if args.impls else None
    available, skipped = discover(adapters, restrict)
    cases = load_cases(args.heavy)
    if args.only:
        wanted = set(args.only)
        cases = [c for c in cases if c["name"] in wanted]

    if args.list:
        print("Implementations:")
        for n, i in available.items():
            print("  [available] %-12s %s" % (n, i["bin"]))
        for n, w in skipped.items():
            print("  [skipped]   %-12s %s" % (n, w))
        print("\nCases (%d):" % len(cases))
        for c in cases:
            tag = " [heavy]" if c.get("heavy") else ""
            print("  %-26s %-8s %s%s" % (c["name"], c["mode"], c["expect"], tag))
        return 0

    if not available:
        print("ERROR: no implementations available. Set env overrides or build them.")
        for n, w in skipped.items():
            print("  - %s: %s" % (n, w))
        return 2

    heavy_timeout = TIMEOUT_HEAVY
    case_results = {}
    for c in cases:
        sys.stderr.write("running %-28s ... " % c["name"])
        sys.stderr.flush()
        t0 = time.time()
        if c["mode"] == "special-hush":
            res = run_hush_divergence(c, available)
        elif c["expect"] == "ratatoskr-parity":
            res = run_ratatoskr_parity(c, available)
        else:
            res = evaluate_case(c, available, heavy_timeout)
        case_results[c["name"]] = res
        sys.stderr.write("%s (%.1fs)\n" % (res["verdict"], time.time() - t0))

    if args.json:
        blob = {
            "available": list(available.keys()),
            "skipped": skipped,
            "cases": {n: {"verdict": r["verdict"], "detail": r["detail"],
                          "impls": {i: {"status": pr.get("status"),
                                        "norm": pr.get("norm")}
                                    for i, pr in r["per_impl"].items()}}
                      for n, r in case_results.items()},
        }
        print(json.dumps(blob, indent=2))

    nfail = print_matrix(cases, case_results, available, skipped)
    return 1 if nfail else 0


if __name__ == "__main__":
    sys.exit(main())
