---
name: shen-review
description: Review Shen code and skills for explicit specifications, appropriate functional and logical mechanisms, sound type claims, closure and state behavior, operational limits, and executable evidence. Use after writing Shen, before calling work complete, or when asked to review or refine a Shen program.
---

# Reviewing Shen

Review the claimed behavior and its evidence. These questions are prompts for judgment: a table can be
the specification, a worklist can be the intended search strategy, and an untyped representation can be
a deliberate boundary. Explain the choice and check its obligations.

## 1. Specification and representation

- Are the domain rules or source facts present, or only a hand-written sample of their outputs?
- Which source is authoritative when facts have several views? Check extension behavior as well as the
  initial examples. A course catalog can contain isolated courses absent from a prerequisite graph;
  deriving the catalog from edge endpoints would lose information.
- Does the representation support the operations the task needs? Use structure for structural work,
  strings for text, and tags where they distinguish meaningful alternatives.
- Does the program claim only valid outputs, or all outputs, unique outputs, ordering or termination?
  A `(list T)` result proves none of those additional properties by itself.
- If a table or prepared closure is derived, what connects it to the source specification and when is
  it rebuilt? Test an extension that would expose a stale view.

## 2. Functional decomposition and engine choice

- Can a common computation be factored into a small higher-order function? Are partial application and
  `/.` clearer than another interpreter, macro or global variable?
- For relations and search, did the design consider `defprolog` and its goal order? A visited set is
  still appropriate when cycles must be bounded; choosing Prolog does not supply termination.
- For token recognition, did it consider typed `defcc`, semantic results and complete consumption?
- If rules are data, does the interpreter separate rule representation from evaluation policy and
  rendering? If they are dynamic relations, would Prolog's dynamic database fit?
- Does a macro need control over syntax, bindings or evaluation? A function suffices for ordinary
  runtime value composition. See `shen-logic-grammars` for expansion and helper-loading phases.

## 3. Closures, state and observation

- What does each closure capture: fixed configuration, a persistent value, or mutable storage?
- Is preparation intentionally outside or inside the lambda? Moving it changes frequency, timing,
  sharing and freshness; it can also change behavior when the preparation has effects.
- Do aliases share state? Do separate factory calls allocate independent state? Test both when relevant.
- Would explicit state and returned effect descriptions make replay or comparison easier? Capturing
  policy is compatible with passing evolving state explicitly.
- If an index describes length or permissions, can mutation through an alias invalidate that fact?
  Persistent stack operations preserve the original value; a destructive implementation needs its own
  account of aliasing and admissible uses.
- Can clients make every required observation? Abstract values can be stored in typed collections and
  passed to generic operations. Printing does not promise a portable encoding; closure equality does
  not establish behavioral equivalence. Provide named observations or a data representation when needed.

See `shen-thinking`'s references on closures, state and indexed stacks.

## 4. Types and proof obligations

- Do signatures connect the inputs and outputs that must agree? Would an invariant remove repeated
  checks or state a useful internal preservation property?
- Are concrete datatype introductions and eliminations both available where needed? A double-line
  rule supplies both directions; an introduction alone need not reveal a representation's components.
- Is a runtime test connected to a `verified` premise when the proof depends on its result? Validate
  the bridge predicate's meaning; a trusted rule can make a misleading assertion look well typed.
- Are boundary validity, transformation preservation and final observations treated separately?
- Are `declare`, operation axioms and untyped definitions together and explained? Client typechecking
  under an asserted interface is not a proof of the implementation's laws.
- Are tags and refinements doing useful work? Internal invariants can be valuable; do not restrict all
  refinements to external boundaries or eliminate tags merely because a list is available.
- Does any claimed property exceed the judgment actually checked? Ordinary arrow types do not imply
  purity, termination, absence of I/O, exclusive ownership or a temporal usage protocol.

## 5. Proof search and phase mistakes

Use `shen-types` when checking rejects unexpectedly, loops or appears to prove too much:

- Distinguish a type variable to solve from an input pattern to match. Use modes where a rule must
  inspect an established union/index instead of inventing one through unification.
- Check goal and premise order, cuts, recursive subtype cycles and left rules that recreate their own
  hypothesis. An unexplained inference explosion calls for inspecting proof search.
- Guard executable side conditions against variables and values outside their domain. A helper defined
  earlier in the same file can serve a later datatype's proof-time side condition. A macro helper needed
  while reading the file must already be loaded; these are different phases.
- Avoid accidental reuse of built-in type names such as `unit`, Shen's unknown-data boundary type.
- Separate symbol data from callable values: use `(fn F)` for an unapplied named function. Do not infer
  that `+` ceases to be a symbol because direct constant patterns encounter a checker limitation.
- A heterogeneous list needs a shared element type; it is not inherently untypable.
- A typed grammar guard needs an extra refinement rule only if its successful test must establish a
  narrower result type. Ordinary well-typed guards do not all need `refine-if`.
- Structural indices do not supply every induction automatically. State the needed lemma or operation
  judgment and test its legal and illegal clients.

## 6. Operational behavior

- **Prolog:** document useful input/output modes, ground negation-as-failure, the closed-world reading,
  goal order, duplicate answers, cycles and search limits. Test an isolated fact and a cyclic graph if
  the domain permits them.
- **Grammars:** check trailing input, alternative order, limited backtracking, left recursion, nullable
  recursion, precedence and associativity. Keep semantic actions pure when they may be retried.
- **Generators:** enforce the depth domain and explain what the bound counts. Depth does not bound the
  number of outputs. Test empty and nullable productions, including rendering on both sides of epsilon.
- **Macros:** inspect the expansion, fixed-point interactions, package qualification, fresh binders and
  evaluation count. Load expansion-time helpers before reading clients that need them.

## 7. Evidence before completion

- Load typed files with checking enabled first:
  `bifrost eval -e '(do (tc +) (load "prog.shen"))'`. State deliberate untyped loads separately.
- Assert predicted results and operation laws; merely evaluating an expression does not check its value.
- Pair a valid client with an isolated invalid client for each enforced rule. Require a type error for
  the intended client; a missing dependency or unrelated error is not success.
- Use small differential checks when two representations should have the same observations. Sampled
  agreement supports testing; it is not a universal representation-independence proof.
- Report the port/kernel, tested boundary and remaining trusted assumptions. Run
  `bifrost skills check --dir skills --impl PORT` for skill edits, including their reference examples.
