# Suite manifests

Any Shen project can run *its* tests across every Bifrost port.

```bash
bifrost --suite /path/to/project/bifrost.suite.json --heavy
go run ../bifrost --suite ./bifrost.suite.json --heavy
```

Ports still come from Bifrost's [`adapters.json`](../adapters.json). The
project supplies only where sources live and how success is reported.

```jsonc
{
  "name": "my-project",
  "root": ".",
  "programs_dir": ".",
  "default_cwd": ".",
  "strip_line_prefixes": ["my loader banner"],
  "cases": [
    {
      "name": "self-suite",
      "mode": "script",
      "program": "load.shen",
      "expect": "agreement",
      "marker": "ALL PASS",
      "heavy": true,
      "doc": "all ports agree and report ALL PASS"
    }
  ]
}
```

`cases_dir` may point at a directory of `*.json` files instead of inline
`cases`.

The script entrypoint should load sources, print **deterministic** output, and
print a **marker** line iff everything passed.

| `expect` | `marker` | assertion |
|----------|----------|-----------|
| `agreement` | set | stdout identical across ports **and** contains the marker |
| `agreement` | — | identical across ports |
| `marker` | set | each port prints the marker; ports need not agree |
| `output` | optional | equals `golden` (+ marker if set) |

Worked example: [`examples/tiny-suite/`](../examples/tiny-suite/).
`--shake` on a suite runs script-mode cases through the deploy-path pipeline.
