# Trial report

Command actually run from the trial directory:

```sh
ls input-skills; find input-skills -type f; echo "$BIFROST_TRIAL_BIN"
$BIFROST_TRIAL_BIN skills check --impl shen-jit --dir answers
```

The checker reported **8 blocks, 0 failed (shen-jit)**. It checked each executable fence in a fresh process with the earlier accepted fences in its file; the review is prose and has no executable fence. No dependencies were installed or runtime/checker changed. There were no failed checker attempts.

- **vocabulary — complete for the stated examples.** Typed Cartesian-product generator over source lists, with epsilon-aware rendering. Exact four and six result lists, uniqueness, and epsilon cases are asserted. Completeness depends on finite source lists; output uniqueness additionally depends on injective rendering for the vocabulary. No trusted declarations or untyped boundaries.
- **history — complete for the requested interface.** Checked cons-based indexed representation and operation bodies, including a length-preserving numeric double. The test observes the old newest value, replacement, and both predecessors. The checker rejected the named false-growth implementation and the two empty-read clients. Concrete datatype rules are part of the trusted type theory; no unchecked operation declarations. This is not a formal proof of all semantic laws or purity.
- **prerequisites — complete for the ground finite graph.** A typed visited worklist expands each discovered job, with the starting job previsited. Tests assert exact results for publish, seed, and isolated docs. Ground membership, not Prolog negation, controls revisits. Termination and completeness are argued from this finite graph, not established by types. No declarations or untyped boundaries.
- **parser — complete for the requested flat token language.** Typed Shen-YACC semantic actions construct an AST with left-folding tails and precedence, then a checked evaluator computes its value. Tests cover precedence, repeated operators and exact associativity, empty and missing-operand input, and trailing tokens. The `number?` verified bridge and conditional refinement are trusted datatype rules; no untyped parse-result declaration. The typechecker does not prove grammar completeness, termination, lexer behavior, or general semantic equivalence; tests are finite.
- **transition — complete for the specified behaviors under documented nonnegative input assumptions.** A configured closure captures the fee, with explicit balance/action and tagged result; a separate closure captures a freshly allocated mutable vector. Assertions cover configurations, replay, unchanged input, credit, overdraft, aliases and distinct factories. The nonnegative domain is an assumption, not enforced. No declarations or untyped boundaries. Effects, freshness and replay are behavioral claims about bodies/runtime, not arrow-type guarantees.
- **review — complete as a prose review.** The minimal typed client exposes the unjustified push axiom without inventing a runtime result. Repair and evidence strategy distinguish checked bodies, trusted rules and semantic properties. No executable claims or trusted declarations.
