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
    python3 bifrost.py --suite M.json   # run a third-party project's suite
                                        #   manifest instead of Bifrost's corpus

Bifrost is also a reusable FRAMEWORK: any Shen project can drop a
`bifrost.suite.json` manifest in its repo and have its own test suite run
across every supported port (see build_suite_from_manifest / run_suite, and
the README section "Using Bifrost from another Shen program").

And it is the single FRONT DOOR for Shen across all ports (Roswell-style),
via subcommands that reuse the same adapters.json port registry:
    bifrost run FILE [--impl X]     # run a .shen program on a chosen/active port
    bifrost eval -e EXPR [--impl X] # evaluate an expression
    bifrost repl [--impl X]         # interactive REPL
    bifrost impls [--versions]      # list ports (state, live kernel, status)
    bifrost use IMPL [--project]    # select the active port (global or ./.bifrost-impl)
    bifrost install IMPL            # install a port via its backend (brew/npm/luarocks/git-build)
    bifrost build FILE OUT --target T   # standalone artifact (delegates to Ratatoskr)

Exit code is non-zero on a real FAILURE. KNOWN DIVERGENCES are reported in
their own section and do NOT, by themselves, fail the run.
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
CASES_DIR = os.path.join(HERE, "cases")
PROGRAMS_DIR = os.path.join(HERE, "programs")


# --------------------------------------------------------------------------
# Cross-platform helpers
# --------------------------------------------------------------------------
# Bifrost runs on Linux, macOS and Windows. The launchers it drives differ by
# OS: native ports compile to `shen.exe` on Windows, and some ports ship a
# `.bat`/`.cmd` wrapper that CreateProcess cannot launch directly. These helpers
# are parameterised (platform/env args) so the Windows code paths are unit-tested
# on every OS, not just on a Windows box. `sys.platform` is "win32" on Windows.

IS_WINDOWS = (os.name == "nt")


def user_config_dir(app="bifrost", platform_name=None, env=None):
    """Per-user config directory for `app`, OS-appropriate.

    Windows -> %APPDATA%\\<app>;  else -> $XDG_CONFIG_HOME/<app> or ~/.config/<app>.
    """
    env = os.environ if env is None else env
    platform_name = platform_name or sys.platform
    if platform_name == "win32":
        base = env.get("APPDATA") or os.path.join(os.path.expanduser("~"), "AppData", "Roaming")
    else:
        base = env.get("XDG_CONFIG_HOME") or os.path.join(os.path.expanduser("~"), ".config")
    return os.path.join(base, app)


def _pathext(env=None):
    env = os.environ if env is None else env
    return [e.lower() for e in env.get("PATHEXT", ".COM;.EXE;.BAT;.CMD").split(";") if e.strip()]


def find_executable_path(path, is_windows=None, env=None):
    """Return an existing executable for `path`, or None.

    The path itself if it exists; otherwise, on Windows, `path` + each PATHEXT
    extension (default_paths are written POSIX-style/extensionless, but the real
    launcher on Windows is `shen.exe` / `shen.cmd`).
    """
    is_windows = IS_WINDOWS if is_windows is None else is_windows
    if os.path.exists(path):
        return path
    if is_windows:
        for ext in _pathext(env):
            if os.path.exists(path + ext):
                return path + ext
    return None


def wrap_executable(argv, is_windows=None):
    """Make argv[0] runnable on this platform.

    On Windows a `.bat`/`.cmd` cannot be launched directly via the default
    (shell-less) subprocess path, so wrap it in `cmd /c`; a `.sh` launcher is
    wrapped in `sh` (needs git-bash/WSL/MSYS `sh` on PATH). A no-op everywhere
    else and for `.exe`/native launchers.
    """
    is_windows = IS_WINDOWS if is_windows is None else is_windows
    if is_windows and argv:
        low = argv[0].lower()
        if low.endswith((".bat", ".cmd")):
            return ["cmd", "/c"] + list(argv)
        if low.endswith(".sh"):
            return ["sh"] + list(argv)
    return list(argv)


def apply_os_overrides(cfg, platform_name=None):
    """Overlay `cfg['os_overrides'][<platform>]` onto `cfg` (shallow merge).

    Lets adapters.json carry OS-specific `default_paths` / `launcher` / templates
    (keys "win32" / "darwin" / "linux") without forking the whole adapter.
    """
    platform_name = platform_name or sys.platform
    ov = (cfg.get("os_overrides") or {}).get(platform_name)
    if not ov:
        return cfg
    merged = dict(cfg)
    merged.update(ov)
    return merged


def find_adapters_path():
    """Locate adapters.json. Resolution order (first existing wins):

      1. $BIFROST_ADAPTERS                          (explicit override)
      2. ./adapters.json in the current directory   (per-project config)
      3. the copy shipped next to this module        (packaged default)

    Launcher locations are machine-specific, so a globally-installed tool
    (uvx/pipx) almost always wants 1 or 2 -- but the packaged default keeps
    `uvx bifrost` working out of the box for the original side-by-side layout.
    """
    env = os.environ.get("BIFROST_ADAPTERS")
    if env and os.path.isfile(env):
        return os.path.abspath(env)
    cwd_cfg = os.path.join(os.getcwd(), "adapters.json")
    if os.path.isfile(cwd_cfg):
        return cwd_cfg
    return os.path.join(HERE, "adapters.json")


ADAPTERS_PATH = os.path.join(HERE, "adapters.json")  # packaged default

# Order in which impls appear in the matrix.
IMPL_ORDER = ["shen-cl", "shen-go", "shen-rust", "shen-lua", "ShenScript", "shen-scheme", "shen-julia"]

TIMEOUT_DEFAULT = 60      # seconds, per invocation
TIMEOUT_HEAVY = 300       # seconds, for heavy (ratatoskr) cases


# --------------------------------------------------------------------------
# Suites: a suite bundles the cases to run with where their programs live and
# the cwd to run them from. The built-in DEFAULT_SUITE is Bifrost's own
# self-test corpus; a third-party Shen project supplies its own suite via a
# `bifrost.suite.json` manifest (see build_suite_from_manifest / --suite).
# --------------------------------------------------------------------------

# Bifrost's own corpus. `default_cwd=None` -> launchers inherit Bifrost's cwd;
# the built-in script-mode programs use absolute {file} paths so cwd is moot.
DEFAULT_SUITE = {
    "name": "bifrost",
    "programs_dir": PROGRAMS_DIR,
    "cases_dir": CASES_DIR,
    "inline_cases": None,
    "root": HERE,
    "default_cwd": None,
    "extra_strip_prefixes": (),
}


def build_suite_from_manifest(path):
    """Build a suite dict from a `bifrost.suite.json` manifest.

    Manifest schema (all paths relative to the manifest file's directory):

        {
          "name": "shen-cas",
          "root": ".",                 # project root; default = manifest dir
          "programs_dir": ".",         # where script {file} programs live
                                       #   (default = root)
          "cases_dir": "bifrost/cases",# optional dir of *.json corpus files
          "cases": [ ...case dicts... ],# optional inline cases (preferred)
          "default_cwd": ".",          # cwd to run launchers from
                                       #   (default = root)
          "strip_line_prefixes": [ "shen-cas loaded" ]  # extra normaliser
                                       #   prefixes to drop as chatter
        }

    Exactly one of `cases` / `cases_dir` is normally given; `cases` wins.
    """
    path = os.path.abspath(path)
    base = os.path.dirname(path)
    with open(path) as f:
        m = json.load(f)

    def resolve(rel, default):
        if rel is None:
            return default
        return os.path.abspath(rel if os.path.isabs(rel) else os.path.join(base, rel))

    root = resolve(m.get("root"), base)
    programs_dir = resolve(m.get("programs_dir"), root)
    cases_dir = resolve(m.get("cases_dir"), None) if m.get("cases_dir") else None
    default_cwd = resolve(m.get("default_cwd"), root)
    return {
        "name": m.get("name", os.path.basename(base) or "suite"),
        "programs_dir": programs_dir,
        "cases_dir": cases_dir,
        "inline_cases": m.get("cases"),
        "root": root,
        "default_cwd": default_cwd,
        "extra_strip_prefixes": tuple(m.get("strip_line_prefixes", [])),
        "adapters": resolve(m.get("adapters"), None) if m.get("adapters") else None,
    }


# --------------------------------------------------------------------------
# Output normalisation
# --------------------------------------------------------------------------
# Cross-impl stdout carries impl-specific noise. We strip it so that *behaviour*
# is compared, not banner chrome. The rules below are deliberately conservative
# and each is justified by an observed, real divergence in launcher chatter
# (NOT in computed values).

def normalize(text, extra_prefixes=()):
    """Normalise an impl's stdout for comparison.

    - strip a trailing run-time banner line ("run time: ... secs") emitted by
      the kernel `load`/timer on several ports,
    - strip `(fn NAME)` declaration echoes that `load` prints (shen-lua's file
      path is `(load FILE)`, which echoes every define),
    - strip lone `0`/`nil`/`true` load-echo lines that `load` prints as the
      value of a trailing `(nl)` form (shen-lua),
    - strip any line whose stripped form starts with one of `extra_prefixes`
      (suite-supplied; lets a third-party project drop its own per-port chatter
      such as a loader banner without forking this normaliser),
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
        if any(stripped.startswith(p) for p in extra_prefixes):
            continue
        out.append(s)
    # drop leading/trailing blanks
    while out and out[0].strip() == "":
        out.pop(0)
    while out and out[-1].strip() == "":
        out.pop()
    # shen-lua's `load` echoes the VALUE of the final top-level form after the
    # program's own output: `0` for a trailing `(nl)`, or `true`/`false`/`nil`
    # for an entrypoint that ends in a boolean result (the common shape for a
    # test harness, e.g. shen-cas's `(run-all-tests)`). If the last surviving
    # line is one of those lone echo tokens preceded by real content, drop it.
    # Other ports don't emit this echo, so the strip is a no-op for them unless
    # their real output legitimately ends in a bare token line (rare; the
    # >= 2 guard preserves single-value eval results such as `true`).
    if len(out) >= 2 and out[-1].strip() in ("0", "true", "false", "nil"):
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

def load_adapters(path=None):
    """Load the per-impl adapter config. Defaults to Bifrost's adapters.json.

    A third-party suite may point at its own adapters file, but the built-in
    one is usually correct: launcher locations are machine-global, not
    project-specific. With no explicit path, resolution follows
    find_adapters_path() ($BIFROST_ADAPTERS, ./adapters.json, packaged copy).
    """
    with open(path or find_adapters_path()) as f:
        raw = json.load(f)
    return {k: v for k, v in raw.items() if not k.startswith("_")}


def resolve_bin(name, cfg):
    """Resolve an impl's launcher path. Returns absolute path or None.

    Order: env var override, then first existing default_path. Relative
    default paths are resolved against the bifrost repo root (HERE). On Windows,
    extensionless candidates also match `shen.exe` / `shen.cmd` (PATHEXT), and
    any `os_overrides` for the platform are applied first.
    """
    cfg = apply_os_overrides(cfg)
    env = cfg.get("env")
    if env and os.environ.get(env):
        hit = find_executable_path(os.environ[env])
        if hit:
            return os.path.abspath(hit)
    for p in cfg.get("default_paths", []):
        cand = p if os.path.isabs(p) else os.path.join(HERE, p)
        hit = find_executable_path(cand)
        if hit:
            return os.path.abspath(hit)
    return None


def discover(adapters, restrict=None):
    """Return (available, skipped). available: name->{cfg,bin}.

    The stored cfg has OS overrides applied, so launch templates reflect the
    current platform (e.g. a Windows-specific `launcher`/`eval` template).
    """
    available, skipped = {}, {}
    for name in IMPL_ORDER:
        if name not in adapters:
            continue
        if restrict and name not in restrict:
            continue
        cfg = apply_os_overrides(adapters[name])
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

def run_invocation(argv, timeout, stdin_eof=False, cwd=None):
    """Run argv, capture combined stdout (+stderr). Returns dict.

    `cwd`, when given, is the working directory the launcher runs in. This
    matters for third-party suites whose entrypoint does relative `(load
    "src/...")` — Shen's `load` resolves paths against the process cwd, so the
    suite must run from its own project root.
    """
    argv = wrap_executable(argv)
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
            cwd=cwd,
        )
        return {"rc": proc.returncode,
                "out": proc.stdout.decode("utf-8", "replace"),
                "timeout": False}
    except subprocess.TimeoutExpired as e:
        partial = e.stdout.decode("utf-8", "replace") if e.stdout else ""
        return {"rc": None, "out": partial, "timeout": True}
    except FileNotFoundError as e:
        return {"rc": None, "out": "launcher missing: %s" % e, "timeout": False}


def build_argv(impl, case, programs_dir=PROGRAMS_DIR):
    """Build the argv for (impl, case) according to its mode.

    `programs_dir` is where script-mode `{file}` programs live (the built-in
    corpus's `programs/` by default; a third-party suite supplies its own).
    """
    cfg = impl["cfg"]
    binpath = impl["bin"]
    subs = {"{bin}": binpath}
    mode = case["mode"]
    if mode == "eval":
        subs["{expr}"] = case["expr"]
        tmpl = cfg["eval"]
    elif mode == "script":
        prog = case["program"]
        prog = prog if os.path.isabs(prog) else os.path.join(programs_dir, prog)
        subs["{file}"] = os.path.abspath(prog)
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


def run_case_on_impl(name, impl, case, heavy_timeout, suite=None):
    suite = suite or DEFAULT_SUITE
    timeout = heavy_timeout if case.get("heavy") else TIMEOUT_DEFAULT
    mode = case["mode"]
    programs_dir = suite["programs_dir"]
    extra = suite.get("extra_strip_prefixes", ())
    # Per-case cwd wins, else the suite default. Relative cwds resolve against
    # the suite root so a manifest can say `"cwd": "."`.
    cwd = case.get("cwd", suite.get("default_cwd"))
    if cwd and not os.path.isabs(cwd):
        cwd = os.path.join(suite.get("root", HERE), cwd)
    cwd = os.path.abspath(cwd) if cwd else None

    if mode == "version":
        argv = build_argv(impl, case, programs_dir)
        r = run_invocation(argv, timeout, cwd=cwd)
        out = normalize(r["out"], extra)
        ok = (not r["timeout"]) and (case["contains"] in r["out"])
        return {"raw": r["out"], "norm": out, "ok": ok, "timeout": r["timeout"], "rc": r["rc"]}

    if mode == "repl-eof":
        argv = build_argv(impl, case, programs_dir)
        r = run_invocation(argv, timeout, stdin_eof=True, cwd=cwd)
        # success = clean exit (no hang). rc 0 expected; any non-timeout exit ok.
        ok = (not r["timeout"])
        return {"raw": r["out"], "norm": normalize(r["out"], extra), "ok": ok,
                "timeout": r["timeout"], "rc": r["rc"]}

    # eval / script
    argv = build_argv(impl, case, programs_dir)
    r = run_invocation(argv, timeout, cwd=cwd)
    norm = normalize(r["out"], extra)
    return {"raw": r["out"], "norm": norm, "ok": None, "timeout": r["timeout"], "rc": r["rc"]}


# --------------------------------------------------------------------------
# Cases
# --------------------------------------------------------------------------

def load_cases(include_heavy, cases_dir=None, inline_cases=None):
    """Collect cases for a run.

    Two sources, used in this precedence:
      * `inline_cases` -- a list of case dicts embedded directly in a suite
        manifest (each may carry its own `group`); preferred when present.
      * `cases_dir` -- a directory of `*.json` corpus files (the built-in
        layout: each file is `{"group": ..., "cases": [...]}`).
    Heavy cases are dropped unless `include_heavy`.
    """
    cases = []
    if inline_cases is not None:
        for c in inline_cases:
            c.setdefault("group", "suite")
            c.setdefault("_file", "<manifest>")
            if c.get("heavy") and not include_heavy:
                continue
            cases.append(c)
        return cases
    cases_dir = cases_dir or CASES_DIR
    for fn in sorted(os.listdir(cases_dir)):
        if not fn.endswith(".json"):
            continue
        with open(os.path.join(cases_dir, fn)) as f:
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

def _marker_check(results, marker):
    """Mark FAIL any impl whose normalised output lacks `marker`.

    Returns (all_present, missing_names). Used by `output`/`agreement` cases
    that carry a `marker` field and by the standalone `marker` expect: a
    third-party suite typically prints a final token such as `ALL PASS`, and
    every port must emit it for the run to be green.
    """
    missing = []
    for n, r in results.items():
        if r.get("timeout") or marker not in r["norm"]:
            missing.append(n)
    return (not missing), missing


def evaluate_case(case, available, heavy_timeout, suite=None):
    """Run case on every available impl and decide pass/fail/divergence.

    Returns dict:
      per_impl: name -> result (with .norm, .status in PASS/FAIL/DIVERGE/SKIP)
      verdict : "PASS" | "FAIL" | "DIVERGE"
      detail  : human string
    """
    results = {}
    for name, impl in available.items():
        results[name] = run_case_on_impl(name, impl, case, heavy_timeout, suite)

    expect = case["expect"]
    marker = case.get("marker")
    verdict = "PASS"
    detail = ""

    if expect == "marker":
        # Self-judging mode: every port must emit `marker` (e.g. the suite's
        # own "ALL PASS") and exit cleanly. Output need NOT agree across ports.
        ok, missing = _marker_check(results, marker)
        for n, r in results.items():
            r["status"] = "FAIL" if n in missing else "PASS"
        if not ok:
            verdict = "FAIL"
            detail = "marker %r missing on: %s" % (marker, ", ".join(missing))
        return {"per_impl": results, "verdict": verdict, "detail": detail}

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
            # All ports agree. If a marker is required, the (identical) output
            # must also contain it -- this is the "agree + pass marker" mode a
            # third-party suite uses: byte-identical across ports AND green.
            if marker is not None:
                ok, missing = _marker_check(results, marker)
                if ok:
                    verdict = "PASS"
                    for r in results.values():
                        r["status"] = "PASS"
                else:
                    verdict = "FAIL"
                    for r in results.values():
                        r["status"] = "FAIL"
                    detail = ("ports AGREE but marker %r absent (suite did not "
                              "report success on any port)" % marker)
            else:
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
        argv = wrap_executable(argv)
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
# Shake-then-run: shake a script-mode program via Ratatoskr, build a standalone
# artifact for each target, run it, and diff the artifacts against each other.
# This exercises the real stand-alone DEPLOY path (not load-from-source), so it
# is the strongest form of cross-impl agreement. Opt-in via --shake.
# --------------------------------------------------------------------------

# Each Ratatoskr build target maps onto the impl column it runs on, so shake
# results line up with the normal matrix. Targets with no builder yet
# (shen-scheme, shen-julia) simply show SKIP in their columns.
TARGET_TO_IMPL = {"lisp": "shen-cl", "lua": "shen-lua", "go": "shen-go",
                  "rust": "shen-rust", "js": "ShenScript"}


def _import_ratatoskr_cli():
    """Import the ratatoskr_cli module from the Ratatoskr repo. Returns the
    module or None if the repo / wrapper is not present."""
    rtk_dir = os.environ.get("BIFROST_RATATOSKR_DIR", RATATOSKR_DIR_DEFAULT)
    mod_path = os.path.join(rtk_dir, "ratatoskr_cli.py")
    if not os.path.isfile(mod_path):
        return None
    if rtk_dir not in sys.path:
        sys.path.insert(0, rtk_dir)
    try:
        import ratatoskr_cli
        return ratatoskr_cli
    except Exception:
        return None


def _launcher_argv(impl):
    cfg = impl["cfg"]
    return list(cfg.get("launcher", [])) + [impl["bin"]]


def run_shake_case(case, available, suite=None):
    """Shake `case`'s program, build+run a standalone artifact per target, diff.

    Mirrors evaluate_case's return shape (per_impl keyed by IMPL name) so it
    drops straight into the matrix. Honours `marker` like the agreement mode.
    """
    suite = suite or DEFAULT_SUITE
    results = {n: {"raw": "", "norm": "", "ok": None, "timeout": False,
                   "rc": None, "status": "SKIP"} for n in available}
    extra = suite.get("extra_strip_prefixes", ())

    rtk = _import_ratatoskr_cli()
    if rtk is None:
        return {"per_impl": results, "verdict": "PASS",
                "detail": "SKIPPED: Ratatoskr wrapper not found (set "
                          "$BIFROST_RATATOSKR_DIR to a checkout with ratatoskr_cli.py)"}

    prog = case["program"]
    prog = prog if os.path.isabs(prog) else os.path.join(suite["programs_dir"], prog)
    prog = os.path.abspath(prog)

    # Shake once on a host (output is host-independent). Prefer shen-cl.
    host_impl = ("shen-cl" if "shen-cl" in available
                 else next((n for n in available if n != "shen-lua"), None)
                 or next(iter(available), None))
    if host_impl is None:
        return {"per_impl": results, "verdict": "FAIL",
                "detail": "no impl available to host the shake"}
    host = _launcher_argv(available[host_impl])
    style = "positional" if host_impl == "shen-lua" else "sub"
    outdir = tempfile.mkdtemp(prefix="bifrost_shake_")
    try:
        rtk.shake(prog, outdir, host=host, eval_style=style, quiet=True)
    except SystemExit as e:
        return {"per_impl": results, "verdict": "FAIL",
                "detail": "shake failed on host %s: %s" % (host_impl, e)}

    builders = rtk.load_builders()
    timeout = TIMEOUT_HEAVY
    for target, impl_name in TARGET_TO_IMPL.items():
        if target not in builders or impl_name not in available:
            continue  # leaves the column as SKIP
        r = results[impl_name]
        try:
            run_argv = rtk.build(target, outdir)
        except SystemExit as e:
            r.update(status="FAIL", raw="build failed: %s" % e, norm="build-failed")
            continue
        if run_argv is None:
            r.update(status="SKIP", raw="builder tool not on PATH", norm="no-tool")
            continue
        try:
            proc = subprocess.run(wrap_executable(run_argv), stdin=subprocess.DEVNULL,
                                  stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                  timeout=timeout)
            raw = proc.stdout.decode("utf-8", "replace")
            r.update(raw=raw, norm=normalize(raw, extra), rc=proc.returncode, status="PASS")
        except subprocess.TimeoutExpired:
            r.update(status="FAIL", timeout=True, norm="timeout")

    # Decide the verdict over the targets that actually ran.
    ran = {n: r for n, r in results.items() if r["status"] in ("PASS", "FAIL")}
    if not ran:
        return {"per_impl": results, "verdict": "PASS",
                "detail": "SKIPPED: no buildable targets available"}
    if any(r["status"] == "FAIL" for r in ran.values()):
        bad = ", ".join("%s(%s)" % (n, r["norm"]) for n, r in ran.items()
                        if r["status"] == "FAIL")
        return {"per_impl": results, "verdict": "FAIL",
                "detail": "artifact build/run failed: %s" % bad}
    norms = {n: r["norm"] for n, r in ran.items()}
    distinct = set(norms.values())
    marker = case.get("marker")
    if len(distinct) > 1:
        for n in ran:
            results[n]["status"] = "FAIL"
        buckets = {}
        for n, v in norms.items():
            buckets.setdefault(v, []).append(n)
        return {"per_impl": results, "verdict": "FAIL",
                "detail": "shaken artifacts DISAGREE: " + "; ".join(
                    "%r <- %s" % (v, ",".join(ns)) for v, ns in buckets.items())}
    if marker is not None and marker not in next(iter(distinct)):
        for n in ran:
            results[n]["status"] = "FAIL"
        return {"per_impl": results, "verdict": "FAIL",
                "detail": "shaken artifacts agree but marker %r absent" % marker}
    return {"per_impl": results, "verdict": "PASS",
            "detail": "shaken artifacts BYTE-IDENTICAL across %d targets (%s)"
                      % (len(ran), ", ".join(sorted(ran)))}


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
# Execution + programmatic API
# --------------------------------------------------------------------------

def execute_cases(cases, available, suite, heavy_timeout=TIMEOUT_HEAVY,
                  progress=True, shake=False):
    """Run every case across the available impls; return name -> result dict.

    When `shake` is set, script-mode cases are run through the shake-then-run
    pipeline (build + run a standalone artifact per target, diff) instead of
    running the source on each port launcher.
    """
    case_results = {}
    for c in cases:
        if progress:
            sys.stderr.write("running %-28s ... " % c["name"])
            sys.stderr.flush()
        t0 = time.time()
        if c["mode"] == "special-hush":
            res = run_hush_divergence(c, available)
        elif c["expect"] == "ratatoskr-parity":
            res = run_ratatoskr_parity(c, available)
        elif shake and c["mode"] == "script":
            res = run_shake_case(c, available, suite)
        else:
            res = evaluate_case(c, available, heavy_timeout, suite)
        case_results[c["name"]] = res
        if progress:
            sys.stderr.write("%s (%.1fs)\n" % (res["verdict"], time.time() - t0))
    return case_results


def results_blob(available, skipped, case_results):
    """The machine-readable shape emitted by --json and returned by run_suite."""
    return {
        "available": list(available.keys()),
        "skipped": skipped,
        "cases": {n: {"verdict": r["verdict"], "detail": r["detail"],
                      "impls": {i: {"status": pr.get("status"),
                                    "norm": pr.get("norm")}
                                for i, pr in r["per_impl"].items()}}
                  for n, r in case_results.items()},
    }


def run_suite(suite, restrict=None, only=None, include_heavy=False,
              adapters=None, progress=False, shake=False):
    """Run a suite programmatically. Importable entrypoint for other tools.

    `suite` is a dict in the shape of DEFAULT_SUITE / build_suite_from_manifest.
    Returns (blob, case_results, available, skipped, cases) where `blob` is the
    same JSON-able shape as `--json`. Raises nothing for missing impls -- they
    are reported in `skipped`.
    """
    if adapters is None:
        adapters = load_adapters(suite.get("adapters"))
    available, skipped = discover(adapters, restrict)
    cases = load_cases(include_heavy, suite.get("cases_dir"), suite.get("inline_cases"))
    if only:
        wanted = set(only)
        cases = [c for c in cases if c["name"] in wanted]
    case_results = execute_cases(cases, available, suite, progress=progress, shake=shake)
    return results_blob(available, skipped, case_results), case_results, available, skipped, cases


# --------------------------------------------------------------------------
# Front-door subcommands: bifrost as a Roswell-style launcher + impl manager.
#
# These make bifrost the single front door for Shen across all ports, the way
# Roswell (`ros`) is for Common Lisp: `run`/`eval`/`repl` launch a program on a
# chosen/active port, `impls`/`use` list and select, `install` builds/fetches a
# port, `build` shells out to Ratatoskr. The differential test matrix remains
# the DEFAULT (bare `bifrost` / the existing flags); these are added as verbs.
# --------------------------------------------------------------------------

SUBCOMMANDS = ("run", "eval", "repl", "impls", "use", "install", "build")

# Active-impl selection (rustup-style): a per-command override beats a project
# pin beats a global default beats "first available (shen-cl preferred)".
PROJECT_PIN = ".bifrost-impl"
GLOBAL_PIN = os.path.join(user_config_dir("bifrost"), "impl")


def read_pin(path):
    try:
        with open(path) as f:
            v = f.read().strip()
            return v or None
    except OSError:
        return None


def resolve_active_impl(adapters, override=None):
    """Resolve which impl a launch verb should use, and where the choice came
    from. Order: --impl override -> ./.bifrost-impl -> ~/.config/bifrost/impl ->
    default (shen-cl if known, else first in IMPL_ORDER present in adapters).
    Returns (impl_name_or_None, source_str)."""
    if override:
        return override, "--impl"
    proj = read_pin(os.path.join(os.getcwd(), PROJECT_PIN))
    if proj:
        return proj, PROJECT_PIN
    glob = read_pin(GLOBAL_PIN)
    if glob:
        return glob, GLOBAL_PIN
    if "shen-cl" in adapters:
        return "shen-cl", "default"
    for name in IMPL_ORDER:
        if name in adapters:
            return name, "default"
    return None, "default"


def _resolve_one(adapters, name):
    """Resolve a single named impl to {cfg,bin} or exit with a clear message."""
    if name not in adapters:
        sys.stderr.write("bifrost: unknown impl %r (known: %s)\n"
                         % (name, ", ".join(adapters)))
        return None
    binpath = resolve_bin(name, adapters[name])
    if binpath is None:
        sys.stderr.write("bifrost: %s is not installed/built. Try: bifrost "
                         "install %s\n" % (name, name))
        return None
    return {"cfg": apply_os_overrides(adapters[name]), "bin": binpath}


def probe_kernel_version(impl):
    """Best-effort live kernel version for an impl (README badges are stale).
    Uses the `version` template if present, else evals `(version)`."""
    cfg = impl["cfg"]
    if cfg.get("version"):
        case = {"mode": "version", "contains": ""}
    else:
        case = {"mode": "eval", "expr": "(version)"}
    try:
        argv = build_argv(impl, case)
    except Exception:
        return "?"
    r = run_invocation(argv, 30)
    if r["timeout"]:
        return "timeout"
    import re
    m = re.search(r"\b(\d+\.\d+)\b", r["out"])
    return m.group(1) if m else "?"


def cmd_launch(verb, rest, adapters):
    """run / eval / repl — drive one port via its adapters.json templates."""
    ap = argparse.ArgumentParser(prog="bifrost %s" % verb)
    ap.add_argument("--impl", help="port to use (overrides pin/default)")
    ap.add_argument("--raw", action="store_true",
                    help="print launcher output verbatim (skip normalisation)")
    if verb == "run":
        ap.add_argument("file", help=".shen program to run")
    elif verb == "eval":
        ap.add_argument("-e", "--expr", required=True, help="expression to evaluate")
    args = ap.parse_args(rest)

    name, source = resolve_active_impl(adapters, args.impl)
    if name is None:
        sys.stderr.write("bifrost: no impl available; try `bifrost install <impl>`\n")
        return 2
    impl = _resolve_one(adapters, name)
    if impl is None:
        return 2

    if verb == "repl":
        # Interactive: inherit the terminal, don't capture.
        tmpl = impl["cfg"].get("repl") or impl["cfg"]["eval"]
        argv = wrap_executable([_sub_tokens(t, {"{bin}": impl["bin"]}) for t in tmpl])
        sys.stderr.write("# bifrost repl on %s (%s)\n" % (name, source))
        try:
            return subprocess.call(argv)
        except FileNotFoundError as e:
            sys.stderr.write("bifrost: launcher missing: %s\n" % e)
            return 2

    case = ({"mode": "script", "program": os.path.abspath(args.file)} if verb == "run"
            else {"mode": "eval", "expr": args.expr})
    argv = build_argv(impl, case)
    r = run_invocation(argv, TIMEOUT_DEFAULT)
    out = r["out"] if args.raw else normalize(r["out"])
    sys.stdout.write(out + ("\n" if out and not out.endswith("\n") else ""))
    if r["timeout"]:
        sys.stderr.write("bifrost: %s timed out\n" % name)
        return 1
    return r["rc"] or 0


def cmd_impls(rest, adapters):
    """List every known port: availability, live kernel version, status."""
    ap = argparse.ArgumentParser(prog="bifrost impls")
    ap.add_argument("--versions", action="store_true",
                    help="probe each available port's kernel version live (slower)")
    args = ap.parse_args(rest)
    available, skipped = discover(adapters)
    active, source = resolve_active_impl(adapters)
    print("%-13s %-11s %-8s %-12s %s" % ("impl", "state", "kernel", "status", "launcher/why"))
    print("-" * 78)
    for name in IMPL_ORDER:
        if name not in adapters:
            continue
        cfg = adapters[name]
        status = cfg.get("status", "?")
        star = " *" if name == active else ""
        if name in available:
            kern = probe_kernel_version(available[name]) if args.versions else cfg.get("kernel", "-")
            print("%-13s %-11s %-8s %-12s %s%s"
                  % (name, "installed", kern, status, available[name]["bin"], star))
        else:
            print("%-13s %-11s %-8s %-12s %s" % (name, "missing", "-", status, skipped.get(name, "")))
    print("-" * 78)
    print("active impl: %s (%s)%s"
          % (active, source, "" if not args.versions else "  [--versions probed live]"))
    return 0


def cmd_use(rest, adapters):
    """Set the active port (global, or --project for a ./.bifrost-impl pin)."""
    ap = argparse.ArgumentParser(prog="bifrost use")
    ap.add_argument("impl", help="port to make active")
    ap.add_argument("--project", action="store_true",
                    help="pin in ./%s for this project instead of the global default" % PROJECT_PIN)
    args = ap.parse_args(rest)
    if args.impl not in adapters:
        sys.stderr.write("bifrost: unknown impl %r (known: %s)\n"
                         % (args.impl, ", ".join(adapters)))
        return 2
    if args.project:
        path = os.path.join(os.getcwd(), PROJECT_PIN)
    else:
        path = GLOBAL_PIN
        os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(args.impl + "\n")
    print("active impl set to %s (%s)" % (args.impl, path))
    if resolve_bin(args.impl, adapters[args.impl]) is None:
        sys.stderr.write("note: %s is not installed yet — `bifrost install %s`\n"
                         % (args.impl, args.impl))
    return 0


def cmd_build(rest, adapters):
    """Delegate to Ratatoskr: shake a program and build a standalone artifact."""
    ap = argparse.ArgumentParser(prog="bifrost build")
    ap.add_argument("file"); ap.add_argument("outdir")
    ap.add_argument("--target", required=True,
                    help="ratatoskr target (lisp/lua/go/rust/js)")
    ap.add_argument("--run", action="store_true", help="run the artifact after building")
    args = ap.parse_args(rest)
    rtk = _import_ratatoskr_cli()
    if rtk is None:
        sys.stderr.write("bifrost: Ratatoskr not found (set $BIFROST_RATATOSKR_DIR)\n")
        return 2
    rtk.shake(os.path.abspath(args.file), os.path.abspath(args.outdir), quiet=True)
    run_argv = rtk.build(args.target, os.path.abspath(args.outdir))
    if run_argv is None:
        sys.stderr.write("bifrost: target %r needs a toolchain not on PATH\n" % args.target)
        return 3
    if args.run:
        return subprocess.call(wrap_executable(run_argv))
    print("built %s artifact; run with:\n  %s" % (args.target, " ".join(run_argv)))
    return 0


def _run_step(argv, cwd=None, env_extra=None):
    """Run one install step, streaming output. Returns True on success."""
    env = dict(os.environ)
    if env_extra:
        env.update(env_extra)
    argv = list(argv)
    # Resolve a bare command (no path separator) via PATH: on Windows this turns
    # `npm`/`luarocks` into their real `npm.cmd`/`.bat`, which wrap_executable
    # then runs through `cmd /c` (CreateProcess can't launch a bare .cmd).
    if argv and (os.sep not in argv[0] and "/" not in argv[0]):
        resolved = shutil.which(argv[0])
        if resolved:
            argv[0] = resolved
    sys.stderr.write("  $ %s%s\n" % (" ".join(argv), "  (cwd=%s)" % cwd if cwd else ""))
    try:
        return subprocess.call(wrap_executable(argv), cwd=cwd, env=env) == 0
    except FileNotFoundError as e:
        sys.stderr.write("  ! %s\n" % e)
        return False


def cmd_install(rest, adapters):
    """Install a Shen port via its declared backend (brew/npm/luarocks/git-build).

    asdf/mise-style: each port's adapters.json `install` block declares HOW it
    installs. git-build sources are overridable for forks via
    $BIFROST_<IMPL>_GIT / _REF or --git/--ref.
    """
    ap = argparse.ArgumentParser(prog="bifrost install")
    ap.add_argument("impl", help="port to install")
    ap.add_argument("--method", help="override install method "
                    "(brew/npm/luarocks/git-build)")
    ap.add_argument("--git", help="git remote to build from (git-build; for forks)")
    ap.add_argument("--ref", help="git branch/tag to build (git-build)")
    ap.add_argument("--force", action="store_true",
                    help="reinstall even if already available; allow experimental ports")
    args = ap.parse_args(rest)

    name = args.impl
    if name not in adapters:
        sys.stderr.write("bifrost: unknown impl %r (known: %s)\n"
                         % (name, ", ".join(adapters)))
        return 2
    cfg = adapters[name]

    if resolve_bin(name, cfg) is not None and not args.force:
        print("%s is already installed (%s). Use --force to reinstall."
              % (name, resolve_bin(name, cfg)))
        return 0

    spec = cfg.get("install") or {}
    if cfg.get("status") == "experimental" and not args.force:
        sys.stderr.write("bifrost: %s is experimental and may not boot. "
                         "Re-run with --force to try anyway.\n" % name)
        return 2

    method = args.method or spec.get("method")
    if not method:
        sys.stderr.write("bifrost: no install method known for %s. Build it "
                         "manually and point $%s at the launcher.\n"
                         % (name, cfg.get("env", "BIFROST_?")))
        return 2

    # Toolchain precheck — fail loud with the exact missing tool, no partial work.
    for tool in spec.get("needs", []):
        if shutil.which(tool) is None:
            sys.stderr.write("bifrost: %s needs %r on PATH to install via %s. "
                             "Install %r first.\n" % (name, tool, method, tool))
            return 2

    print("installing %s via %s ..." % (name, method))
    ok = False
    if method == "brew":
        ok = _run_step(["brew", "install", spec.get("package", name)])
    elif method == "npm":
        ok = _run_step(["npm", "install", "-g", spec.get("package", name)])
    elif method == "luarocks":
        ok = _run_step(["luarocks", "install", spec.get("package", name)])
    elif method == "git-build":
        ok = _install_git_build(name, cfg, spec, args)
    else:
        sys.stderr.write("bifrost: unknown install method %r\n" % method)
        return 2

    if not ok:
        sys.stderr.write("bifrost: install of %s failed.\n" % name)
        return 1
    # Verify the launcher now resolves.
    if resolve_bin(name, cfg) is None:
        sys.stderr.write("bifrost: %s installed but launcher still not found; "
                         "set $%s to its path.\n" % (name, cfg.get("env", "BIFROST_?")))
        return 1
    print("installed %s -> %s" % (name, resolve_bin(name, cfg)))
    return 0


def _install_git_build(name, cfg, spec, args):
    """Clone (if absent) the port's repo and run its existing `build` recipe.

    The clone target dir is the build recipe's cwd. Remote/ref are overridable
    for forks via --git/--ref or $BIFROST_<IMPL>_GIT / _REF.
    """
    env_key = (cfg.get("env") or "").upper()
    git = args.git or os.environ.get(env_key + "_GIT") or spec.get("git")
    ref = args.ref or os.environ.get(env_key + "_REF") or spec.get("ref")
    build = cfg.get("build")
    if not build or not build.get("cwd"):
        sys.stderr.write("bifrost: %s has no git-build recipe in adapters.json\n" % name)
        return False
    repo_dir = build["cwd"]
    if not os.path.isdir(os.path.join(repo_dir, ".git")):
        if not git:
            sys.stderr.write("bifrost: %s not cloned at %s and no git remote known "
                             "(pass --git URL or set %s_GIT)\n" % (name, repo_dir, env_key))
            return False
        if not _run_step(["git", "clone", git, repo_dir]):
            return False
    if ref and not _run_step(["git", "-C", repo_dir, "checkout", ref]):
        return False
    out = build.get("out", "")
    argv = [_sub_tokens(a, {"{out}": out}) for a in build["argv"]]
    return _run_step(argv, cwd=repo_dir)


def dispatch_subcommand(argv):
    """Route a front-door verb. Returns an exit code, or None if argv[0] is not
    a subcommand (so main() should fall through to the test-matrix parser)."""
    if not argv or argv[0] not in SUBCOMMANDS:
        return None
    verb, rest = argv[0], argv[1:]
    adapters = load_adapters()
    if verb in ("run", "eval", "repl"):
        return cmd_launch(verb, rest, adapters)
    if verb == "impls":
        return cmd_impls(rest, adapters)
    if verb == "use":
        return cmd_use(rest, adapters)
    if verb == "install":
        return cmd_install(rest, adapters)
    if verb == "build":
        return cmd_build(rest, adapters)
    return None


# --------------------------------------------------------------------------
# Main
# --------------------------------------------------------------------------

def main(argv=None):
    raw = sys.argv[1:] if argv is None else argv
    sub = dispatch_subcommand(raw)
    if sub is not None:
        return sub

    ap = argparse.ArgumentParser(description="Bifrost cross-impl Shen meta test harness")
    ap.add_argument("--heavy", action="store_true", help="include heavy (ratatoskr) cases")
    ap.add_argument("--only", nargs="*", help="run only the named cases")
    ap.add_argument("--impls", help="comma-separated subset of impls to drive")
    ap.add_argument("--list", action="store_true", help="list impls + cases and exit")
    ap.add_argument("--json", action="store_true", help="emit machine-readable JSON")
    ap.add_argument("--suite", metavar="MANIFEST",
                    help="run a third-party project's bifrost.suite.json instead "
                         "of Bifrost's own corpus")
    ap.add_argument("--shake", action="store_true",
                    help="for script-mode cases, shake the program via Ratatoskr "
                         "and diff the standalone ARTIFACTS per target (deploy-path "
                         "parity) instead of running the source on each launcher")
    args = ap.parse_args(argv)

    if args.suite:
        suite = build_suite_from_manifest(args.suite)
    else:
        suite = DEFAULT_SUITE
    adapters = load_adapters(suite.get("adapters"))
    restrict = set(args.impls.split(",")) if args.impls else None
    available, skipped = discover(adapters, restrict)
    cases = load_cases(args.heavy, suite.get("cases_dir"), suite.get("inline_cases"))
    if args.only:
        wanted = set(args.only)
        cases = [c for c in cases if c["name"] in wanted]

    if args.list:
        if args.suite:
            print("Suite: %s  (root %s)\n" % (suite["name"], suite["root"]))
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

    case_results = execute_cases(cases, available, suite, shake=args.shake)

    if args.json:
        print(json.dumps(results_blob(available, skipped, case_results), indent=2))

    nfail = print_matrix(cases, case_results, available, skipped)
    return 1 if nfail else 0


if __name__ == "__main__":
    sys.exit(main())
