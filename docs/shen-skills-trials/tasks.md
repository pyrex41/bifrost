# Held-out Shen transfer battery

Read the four supplied `input-skills/*/SKILL.md` files and any linked references you need. Do not read another condition, repository checkout, prior agent session, or answer key. Work only in this trial directory. You may inspect the supplied runtime and run your own programs. Do not delegate, install dependencies, or change the runtime/checker.

Produce solutions for the six tasks below. Write `answers/<task>/SKILL.md` for each, using executable `shen` fences and prose explaining the design, assumptions, and remaining obligations. Keep each task self-contained. Do not copy an example and merely rename it when the requested behavior differs.

Checker contract (identical for both conditions): typed fences are loaded under `(tc +)` and accumulate only within a file. `<!-- reject: full-function-name -->` before a fence expects a type error at that function; rejected fences are not accumulated. `<!-- untyped -->` permits an explicitly explained untyped boundary. Assertions must raise an error on mismatch. A new process checks each fence with its earlier accepted setup. Run `$BIFROST_TRIAL_BIN skills check --impl shen-jit --dir answers`; the runtime is selected by `$BIFROST_SHEN_JIT` and adapters by `$BIFROST_PORTS_ROOT`.

Write `report.md` with per-task result, exact commands actually run, mechanism choice, tests, trusted declarations/untyped boundaries, and properties not established. Report incomplete work and failed attempts honestly. Prefer modest checked solutions to an elaborate unfinished framework. Do not claim improved performance relative to another condition; you cannot see it.

## 1. Vocabulary-driven generation (`vocabulary`)

Define a small authoritative phrase vocabulary with colors `amber`, `violet` and objects `kite`, `cup`. Generate every two-word phrase in color/object order. Extend the vocabulary with `teal` by changing the source vocabulary only, not a hand-maintained derived answer set. Assert the original four results and the extended six, including absence of duplicates. Explain the completeness argument and its assumptions. Support an empty alternative as well, without extra spaces.

## 2. Indexed persistent history (`history`)

Implement a history whose operations are empty, append a newest item, read newest, and remove newest. Track nonemptiness structurally so reading empty and reading after removing the sole item are rejected. Add an operation that replaces the newest numeric item with its double without changing history length or predecessor. Check implementation bodies, not just clients of asserted operation types. Assert persistence, predecessor preservation, and the replacement result. Include a negative implementation that claims growth while returning the old history.

## 3. Cyclic prerequisites (`prerequisites`)

Represent jobs `seed`, `compile`, `package`, `publish`, and isolated `docs`. Compile requires seed; package requires compile; publish requires package; seed also requires package, creating a cycle. Enumerate all transitive requirements for a ground input, excluding the input itself, without duplicates and with a terminating strategy on this finite graph. Assert results for publish, seed and docs. Explain groundness and any use of negation.

## 4. Expression parser (`parser`)

Parse token lists containing numbers, `plus` and `times`, with multiplication precedence and left associativity. Build a typed abstract syntax tree, then evaluate it. Require complete input consumption. Assert a precedence case, repeated operators, and rejection of malformed input and trailing tokens. State what parser behavior or typing does not prove. An untyped trusted parse-result declaration is not a substitute for checked semantic actions.

## 5. Configured transition (`transition`)

Build a configurable account transition that captures an immutable debit fee and takes an explicit current balance plus credit/debit action, returning a new balance or an explicit insufficient-funds result. No hidden account state. Assert two independent configurations, replay with the same inputs, unchanged predecessor balance, and overdraft behavior. Separately demonstrate a private mutable counter: aliases of one factory result share state, separate factories do not. Explain which guarantees are behavioral rather than arrow-type guarantees.

## 6. Review a misleading claim (`review`)

A library implements a stack in untyped closures and then asserts datatype operation rules: `empty : stack A 0`, `push : A --> stack A N --> stack A [succ N]`, and `top : stack A [succ N] --> A`. Its actual push body always returns the original stack. Its author claims: “Clients typecheck, therefore push is correct; length indices also prove ordering, persistence, purity and termination.” Review this claim, propose the smallest counterexample and a repair/evidence strategy, and distinguish checked obligations from assumptions. This task may be prose only; do not invent an execution result.
