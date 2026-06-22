"""Cross-platform unit tests for Bifrost's portability helpers.

These pin the Windows code paths and run identically on Linux, macOS and
Windows — they exercise the *parameterised* helpers (passing platform/env/
is_windows explicitly), so the Windows behaviour is verified on every OS in CI,
not only on a Windows runner. No Shen ports are required.

Run:  pytest test_portable.py
"""

import os

import bifrost


# ---- user_config_dir -----------------------------------------------------

def test_config_dir_windows_uses_appdata():
    got = bifrost.user_config_dir("bifrost", "win32", {"APPDATA": r"C:\Users\me\AppData\Roaming"})
    assert got.startswith(r"C:\Users\me\AppData\Roaming")
    assert got.endswith("bifrost")


def test_config_dir_windows_fallback_without_appdata():
    got = bifrost.user_config_dir("bifrost", "win32", {})
    assert "AppData" in got and "Roaming" in got


def test_config_dir_linux_respects_xdg():
    assert bifrost.user_config_dir("bifrost", "linux", {"XDG_CONFIG_HOME": "/x"}) == os.path.join("/x", "bifrost")


def test_config_dir_posix_default():
    got = bifrost.user_config_dir("bifrost", "darwin", {})
    assert got.endswith(os.path.join(".config", "bifrost"))


# ---- wrap_executable ------------------------------------------------------

def test_wrap_bat_on_windows():
    assert bifrost.wrap_executable([r"C:\p\shen.cmd", "eval"], is_windows=True) == ["cmd", "/c", r"C:\p\shen.cmd", "eval"]
    assert bifrost.wrap_executable([r"C:\p\shen.bat"], is_windows=True) == ["cmd", "/c", r"C:\p\shen.bat"]


def test_wrap_sh_on_windows():
    assert bifrost.wrap_executable(["builders/lisp/build.sh", "a"], is_windows=True) == ["sh", "builders/lisp/build.sh", "a"]


def test_wrap_exe_on_windows_is_noop():
    assert bifrost.wrap_executable([r"C:\p\shen.exe", "eval"], is_windows=True) == [r"C:\p\shen.exe", "eval"]


def test_wrap_noop_on_posix():
    for argv in (["/bin/shen", "eval"], ["x/build.sh"], ["foo.cmd"]):
        assert bifrost.wrap_executable(argv, is_windows=False) == argv


# ---- find_executable_path -------------------------------------------------

def test_find_exact_path(tmp_path):
    f = tmp_path / "shen"
    f.write_text("#!/bin/sh\n")
    f.chmod(0o755)
    assert bifrost.find_executable_path(str(f), is_windows=False) == str(f)


def test_find_non_executable_is_skipped_on_posix(tmp_path):
    # A file that exists but lacks the exec bit (e.g. a committed wrong-arch
    # binary) must NOT be reported as available on POSIX — it would only crash
    # at exec time with "Exec format error". Windows has no exec bit, so there
    # existence alone is enough.
    f = tmp_path / "shen-go"
    f.write_text("not actually runnable")
    f.chmod(0o644)
    assert bifrost.find_executable_path(str(f), is_windows=False) is None
    assert bifrost.find_executable_path(str(f), is_windows=True) == str(f)


def test_find_windows_extension_probe(tmp_path):
    # default_paths are written extensionless; on Windows the real file is shen.exe
    (tmp_path / "shen.exe").write_text("MZ")
    base = str(tmp_path / "shen")
    assert bifrost.find_executable_path(base, is_windows=True) == base + ".exe"
    # ...and on POSIX the extensionless probe must NOT invent the .exe
    assert bifrost.find_executable_path(base, is_windows=False) is None


def test_find_missing_returns_none(tmp_path):
    assert bifrost.find_executable_path(str(tmp_path / "nope"), is_windows=True) is None


# ---- run_invocation resilience -------------------------------------------

def test_run_invocation_handles_exec_format_error(tmp_path):
    # A launcher that exists and is +x but is not a valid executable for this
    # host (e.g. a macOS Mach-O binary on Linux) raises OSError "Exec format
    # error" from the exec path. One unrunnable port must never abort the whole
    # matrix — it should be reported as not runnable, not raised.
    bogus = tmp_path / "wrong-arch"
    # Random bytes that are not a valid script/ELF for the current host.
    bogus.write_bytes(b"\xca\xfe\xba\xbe\x00\x00\x00\x00")
    bogus.chmod(0o755)
    res = bifrost.run_invocation([str(bogus)], timeout=10)
    assert res["rc"] is None
    assert res["timeout"] is False
    assert "not runnable" in res["out"]


def test_run_invocation_missing_launcher(tmp_path):
    res = bifrost.run_invocation([str(tmp_path / "does-not-exist")], timeout=10)
    assert res["rc"] is None
    assert res["timeout"] is False
    assert "launcher missing" in res["out"]


# ---- apply_os_overrides ---------------------------------------------------

def test_os_overrides_merge_for_platform():
    cfg = {"eval": ["a"], "default_paths": ["p"],
           "os_overrides": {"win32": {"eval": ["b"], "launcher": ["luajit"]}}}
    merged = bifrost.apply_os_overrides(cfg, "win32")
    assert merged["eval"] == ["b"]            # overridden
    assert merged["launcher"] == ["luajit"]   # added
    assert merged["default_paths"] == ["p"]   # untouched


def test_os_overrides_noop_for_other_platform():
    cfg = {"eval": ["a"], "os_overrides": {"win32": {"eval": ["b"]}}}
    assert bifrost.apply_os_overrides(cfg, "linux")["eval"] == ["a"]


def test_os_overrides_absent_is_identity():
    cfg = {"eval": ["a"]}
    assert bifrost.apply_os_overrides(cfg, "win32") == cfg


# ---- subcommand router back-compat ---------------------------------------

def test_router_passes_through_non_subcommands():
    # The test-matrix flags must NOT be intercepted by the front-door router.
    assert bifrost.dispatch_subcommand(["--list"]) is None
    assert bifrost.dispatch_subcommand(["--suite", "x.json"]) is None
    assert bifrost.dispatch_subcommand([]) is None


def test_router_recognises_subcommands():
    assert "run" in bifrost.SUBCOMMANDS and "install" in bifrost.SUBCOMMANDS
