# Trial report

## Results

All six requested files are in `answers/<task>/SKILL.md`. The final checker run
reported **10 blocks, 0 failed (shen-jit)**: six accepted typed fences, three
expected type rejections, and one explicitly untyped Prolog fence. Review is
prose only. No failed checker attempts or incomplete implementations occurred.

| Task | Mechanism and evidence | Result / remaining obligations |
|---|---|---|
| vocabulary | Typed Cartesian product over source alternatives, token concatenation, edge rendering. Exact original four and extended six strings; explicit distinctness; empty alternatives on either side, both empty, and no choices. | Passed. Completeness follows by induction on finite slots, not from the shape types. Uniqueness assumes the supplied distinct words and unambiguous rendering. Arbitrary vocabularies may duplicate outputs. |
| history | Structural successor-indexed immutable lists; all five operation bodies checked. Persistence, old/new tops, doubled top, predecessor equality and removal to empty asserted. | Passed, including expected rejection of empty read, read after removing singleton, and dishonest growth implementation. Length types alone do not prove content laws, ordering, purity, or termination. |
| prerequisites | Prolog edge facts plus visited simple-path relation; typed deduplication and set/cardinality assertions for publish, seed, docs. | Passed. Finite facts and ground queries justify termination; simple paths preserve reachability. Output order and unknown-job validation are not promised. |
| parser | Typed `defcc`, token union, AST datatype, checked left-fold actions, typed evaluator. Precedence, left-associated ASTs for both operators, repeated mixed operators, empty/malformed inputs, and trailing tokens asserted. | Passed. No trusted parse-result declaration. Typechecking establishes AST shape, not general language correctness, completeness, ambiguity, termination, or arithmetic laws. |
| transition | Typed immutable-fee closure with explicit balance; separate typed vector-backed counter factory. Two fees, replay/interleaving, credit, unchanged predecessor, insufficient funds, exact exhaustion, shared aliases and fresh factory state asserted. | Both fences passed. Purity/replay, lexical privacy and allocation independence are behavioral/body/runtime properties, not arrow-type theorems. Sequential use only; not decimal finance or concurrent counter semantics. |
| review | Prose identifies the unchecked operation axioms, singleton counterexample, structural repair, negative implementation test and content/model obligations. | Completed review. No execution of the unprovided library or invented outcome. |

Every executable assertion raises an error on mismatch. Parser negative tests
catch parse errors; an assertion failure itself is outside that catch. History
negative fences are expected checker rejections and do not accumulate.

## Trusted surfaces

* Only prerequisites uses an untyped fence: Shen's `defprolog` and its query
  wrapper. `(declare q.raw [symbol --> [list symbol]])` is an explicit trusted
  contract, not a checked implementation theorem. Its callers, deduplication and
  tests are checked. Negation is only ground Boolean membership negation after
  edge binding.
* Parser introduces the `number?` verified refinement and a general verified
  `if` rule for grammar guards. Their logical soundness and the stable meaning
  of `number?` are assumptions; no operation signature is asserted over an
  untyped parser.
* All other operation bodies are typed. Structural datatype rules remain part
  of the specification/trusted logic. No hostile extensions or redefinitions are
  assumed. Primitive arithmetic, closure/list/vector behavior, `compile`, and
  the runtime/checker itself remain trusted.
* Finite examples are behavioral evidence, not universal proofs. No formal
  effect, termination, encapsulation, arithmetic or full parser-correctness
  proof is claimed. No performance comparison was made.

## Commands actually run

All commands ran in
`/private/tmp/bifrost-skills-drive/trials/gpt-6.1-sol-original`.
Environment variables selected the supplied runtime and adapters; none were
changed. The following are the exact shell commands, in chronological order.

```bash
pwd; ls -la; find input-skills -type f; env | grep '^BIFROST_'
```

```bash
"$BIFROST_TRIAL_BIN" skills check --help; "$BIFROST_TRIAL_BIN" eval --help; ls -la "$BIFROST_SHEN_JIT"
```

First check, after writing vocabulary, history, prerequisites and parser:

```bash
"$BIFROST_TRIAL_BIN" skills check --impl shen-jit --dir answers
```

Result: 8 blocks, 0 failed.

Second check, after adding transition and review:

```bash
"$BIFROST_TRIAL_BIN" skills check --impl shen-jit --dir answers
```

Result: 10 blocks, 0 failed. A subsequent prose-only clarification in prerequisites
states that unknown ground symbols also return no requirements.

The four supplied `input-skills/*/SKILL.md` files were read with the file-reading
tool, not shell commands. No linked references were needed. File-writing/editing
tools created the six solutions and this report. No other condition, checkout,
prior session, or answer key was read; no delegation, dependency installation,
or runtime/checker modification was performed.
