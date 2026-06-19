"""Hatchling build hook: compile the Go bifrost binary into the wheel.

Runs `go build` at wheel-build time (so the binary matches the target machine
when uvx/pip builds from sdist or git) and force-includes it as package data.
Requires the Go toolchain on the build machine; the no-Go install paths are
`go install` and the GoReleaser release binaries.
"""

import os
import subprocess
import sys

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class GoBuildHook(BuildHookInterface):
    PLUGIN_NAME = "custom"

    def initialize(self, version, build_data):
        root = self.root
        exe = "bifrost.exe" if sys.platform == "win32" else "bifrost"
        out_dir = os.path.join(root, "pybin", "bifrost_cli", "_bin")
        os.makedirs(out_dir, exist_ok=True)
        out = os.path.join(out_dir, exe)
        # Build the module at the repo root (package main) into the package.
        subprocess.run(
            ["go", "build", "-trimpath", "-o", out, "."],
            cwd=root, check=True,
        )
        # Mark the wheel as platform-specific (it carries a native binary) and
        # ensure the binary is included.
        build_data["pure_python"] = False
        build_data["infer_tag"] = True
        # value is the path within the wheel (sources=["pybin"] already stripped).
        build_data.setdefault("force_include", {})[out] = os.path.join("bifrost_cli", "_bin", exe)
