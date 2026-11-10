# TERRA independent checker review

Reviewed `skills_check.go`, `skills_test.go`, `skills/README.md`, the plan and SOL's checker audit. No repository files changed; no commit or delegation. Content was concurrently changing, so the example count below is a snapshot.

## Findings and gaps

- No additional *demonstrated valid-example* false pass or false fail was established in this pass. In particular, the existing pinned-port regressions verify missing unary/zero-argument helpers, a valid same-block forward typed helper, same-block setup, wrong-definition diagnostic, and isolation.
- The reader-form walker is a conservative syntactic heuristic, not a generic dependency or quotation-aware analysis. It uses runtime arity metadata, does not establish implementation existence, traverses all forms in the failed block including forms after the failure, and re-runs reader macros. The README already describes the arity/declaration and read-time-side-effect trust boundaries, but does not explicitly explain that non-executable forms or later forms can be traversed. Macro-generated or data-like forms and user-defined special syntax remain an important test blind spot. No general fix was attempted without a valid counterexample.
- `reject: ...` matches any whole-token diagnostic phrase, not necessarily the definition name; `reject: type error` can therefore match a different definition. This is documented behavior, not an anchoring guarantee. Authors should use the complete function name; bare `reject` provides no target anchoring.
- Standalone HTML comments are all reserved, even prose comments unrelated to fences. This is intentional fail-closed scope expansion and is explicitly documented; consumers of previously ignored comments may need edits.
- Probe `(define intended {number --> string} X -> (do (quote (missing)) X))` produced `undefined helper quote` during validation. This is **not** evidence of a quoting regression: on this Shen reader `(quote ...)` expands to a call to an undefined function, not a valid quotation form. Likewise `(protect (missing))` was rejected by the checker, but the number-returning control also fails Shen typechecking, so it is not established as a valid negative control. Conversely nested literal data `[[missing]]` in `(define intended {number --> string} X -> (do [[missing]] X))` correctly passed the expected rejection. These failed probes must not be mistaken for confirmed bugs.
- Existing real-port tests do not exercise arbitrary macro expansions, valid quotation/protected data, declarations whose arity exists without an implementation, or every port's diagnostic format. Deliberate marker fabrication, arbitrary changes to typechecking inside a block, and dynamically resolved calls remain outside the checker security/proof contract. No exhaustive adversarial search or portability conclusion is warranted.

## Commands and results

Pinned executable: `/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit`, selected with `BIFROST_SHEN_JIT`.

- `go test -count=1 ./...`: PASS (ordinary invocation skips optional real-port tests).
- `go vet ./...`: PASS.
- `BIFROST_SHEN_JIT=... BIFROST_SKILLS_TEST_IMPLS=shen-jit go test -count=1 -run '^TestSkillsCheckerOnPorts$' -v`: PASS, 22 real-port fixtures.
- `BIFROST_SHEN_JIT=... go run . skills check --impl shen-jit --dir skills`: PASS, 61 blocks, 0 failed.
- Temporary probe fixtures under `/tmp/terra-skills/shen-types/SKILL.md`, executed via `BIFROST_SHEN_JIT=... go run . skills check --impl shen-jit --dir /tmp/terra-skills -v shen-types`: nested literal data PASS; quote and protect probes FAIL as explained above. `shen-jit eval -e '(read-file "/tmp/terra-case.shen")'` confirmed reader expansion. No other port was independently run in this review.
