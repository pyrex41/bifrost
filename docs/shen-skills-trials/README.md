# Exploratory Shen skill transfer trials

These are **artifact-level observations, not evidence of a reproducible or causal improvement**.

## Protocol

The [six-task prompt](tasks.md) probes source-driven generation, indexed history, cyclic reachability,
typed parsing, explicit state versus mutable closures, and review of unjustified proof claims.
Two user-requested models (`gpt-6.1-sol`, `gpt-6.1-terra`) each received one fresh original and one
fresh revised condition through `openai-codex`, high reasoning. Each six-task battery shared a session.
Original skills came from `51a8f0d`; revised inputs were frozen after content cross-review.
All four used the same prompt, checker, official Shen/JIT v0.5 binary (kernel 43.4), 600-second
wall-time allowance, and disabled automatic skills/context/extensions. Two cells ran concurrently.
No cell timed out; input manifests remained unchanged. This was not a sandbox.

The [runner](run.py) stores hashes, command, elapsed time, exit status, timeout and input/output
manifests. Exit 0 is **not** a completion score. Parent independently reran every submitted suite.
Fresh SOL and TERRA reviewers read all 24 answers and session tool calls; review was not blinded.
SOL also tested copied artifacts against additional behavioral oracles.

## Results

| Cell | Independently rerun blocks | Task assessment | Clean-pair eligibility |
|---|---:|---|---|
| SOL original | 10, zero failed | 6 complete | Eligible, qualified |
| SOL revised | 10, zero failed | 6 complete | Eligible, qualified |
| TERRA original | 10, zero failed | 3 complete, 3 partial | **Excluded: external repository source search** |
| TERRA revised | 8, zero failed | 6 complete, one review wording caveat | Eligible individually; no clean original pair |

TERRA original supplied an AST/evaluator but **no token parser**. Its history and account code worked,
but requested retained-predecessor/persistence/replay assertions were incomplete. Its recorded `rg`
searched `$BIFROST_PORTS_ROOT` and returned external checkout source, contrary to instructions.
Its honest partial-parser report does not repair this protocol violation. Do not pool it into an
original-versus-revised improvement estimate.

SOL revised used a checked graph worklist instead of the original's explicitly trusted Prolog boundary
and added a negative semantic-action check; both approaches were appropriate and both completed the
battery. TERRA revised's prose review overstates that an unspecified empty closure “cannot deliver 1”:
it might accidentally return that value. The reviewers differed in whether to flag this wording;
we retain the more conservative caveat. No execution of that hypothetical library occurred.

Independent SOL probes checked each implemented parser against 3,906 token lists of lengths 0–5
plus three longer expressions, all five graph queries, and retained history/account observations.
All tested probes passed. These are finite tests, not universal correctness proofs, and cannot
retroactively supply missing assertions in the submitted artifacts.

## Evidence

[Archived artifacts](../evidence/shen-skills/trials/) retain answers, self-reports, metadata,
independent checker output and extracted tool-call arguments. Local home-directory paths in the
published evidence are redacted as `<HOME>`; executable answers and recorded results
are unchanged. Full raw sessions remain under
`/tmp/bifrost-skills-drive/trials/*/sessions`; the archived call lists are not full transcripts.

- [SOL artifact review and oracle scope](../evidence/shen-skills/trial-review-sol.md)
- [TERRA protocol review](../evidence/shen-skills/trial-review-terra.md)

The additional oracle builder and copies remain in `/tmp/bifrost-sol-review-probes`; their reported
outcomes are attributed to SOL, not independently reproduced parent results. Source snapshots and
raw sessions are local evidence, not guaranteed permanent storage. Final integration changed only
the skills README after the trial snapshot; no trial answer was edited to improve its score.

## Reproduce

Build a checker from this checkout, provide an installed pinned runtime, and choose a **new** output
directory. This starts four paid model sessions by default; it does not run automatically in CI.

```sh
go build -o /tmp/bifrost-trial-checker .
python3 docs/shen-skills-trials/run.py \
  --repo . --baseline 51a8f0d \
  --checker /tmp/bifrost-trial-checker \
  --runtime /absolute/path/to/pinned/shen-jit \
  --ports /absolute/path/to/port/checkouts \
  --output /tmp/new-shen-trials --seconds 600 --jobs 2
```

The runner requires Python 3, git, tar, Pi and configured credentials; it does not modify auth
settings, install models or claim filesystem isolation. Inspect tool calls for protocol violations,
rerun outputs, and score actual requirements rather than block counts.

## Limits and next evaluation

One run per model/condition, correlated tasks in a shared context, no blinding/randomization, and an
excluded baseline prevent a general improvement claim. Revised material is also substantially longer,
so this does not isolate the effect of individual edits. No performance benchmark was run.

Next: enforce filesystem isolation, rerun a clean TERRA pair, repeat both model pairs with independent
seeds/sessions and varied tasks, and preregister scoring. The current evidence supports shipping a
reviewable candidate with bounded claims, **not declaring the transfer-improvement acceptance criterion met**.
