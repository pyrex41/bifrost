---
name: shen-types
description: Design and debug Shen datatype rules, refinements, indexed and abstract types, typed boundaries, and proof search. Use when a Shen invariant should be checked statically, a datatype rejects a valid program or accepts an invalid one, or type checking loops. Includes runnable examples, named rejection cases, modes, and the limits of trusted type axioms.
---

# Design and check Shen types

Treat a `datatype` as a collection of inference rules over Shen objects. The checker compiles these
rules to Prolog and searches for proofs. An ordinary list can inhabit a domain type without a new
runtime wrapper. Start with the invariant the client needs, then choose rules that establish it.

## Workflow

1. State the invariant and one invalid client that should be rejected. Distinguish shape, a relation
   between values, and a semantic law. An arrow signature alone does not prove all three.
2. Choose ordinary signatures for basic shapes. Add datatype rules where they express a useful
   invariant. Give domain types fresh names: Shen's existing `unit` means an object with no usable
   type information; it is not a name for physical units.
3. Separate syntax inspected during checking from values computed at runtime. Use side conditions
   for the former, and a guarded boundary plus a justified `verified` rule for the latter.
4. Supply introduction and elimination rules deliberately. Trace the failing judgment before adding
   an axiom. Read [proof search](references/proof-search.md) if a proof loops or needs an annotation.
5. Mark every trusted rule and implementation boundary. The checker proves consequences of the
   active theory. It does not independently establish that user axioms are sound, predicates retain
   their meaning, functions terminate, or mutable state preserves an invariant.
6. Enable `(tc +)` **before** loading typed files. Run positive examples with value assertions and
   isolate each invalid client in a named `define`. Check that its diagnostic names that definition;
   an unrelated type error is not evidence that the intended invariant holds.

Datatypes are open: later rules can admit more objects. Keep the active theory and any assumptions
about exhaustive cases under control. A catch-all case is appropriate only when it is correct for
every remaining object admitted by that theory.

## Read a rule

| Form | What it does | What to check |
|---|---|---|
| `if Test` | Evaluates a side condition on the term being checked | Make it total on unexpected syntax; it does not inspect a future runtime value |
| `X : T;` | Searches for a proof that `X` has type `T` | It can use hypotheses about variables |
| `Hyps >> Goal;` | Proves a goal with the stated hypotheses | Elimination rules must make progress |
| `____` | Introduces one rule | The conclusion determines whether it is a right or left rule |
| `====` | Generates introduction and elimination rules | Use it when a constructed value must later reveal its components |
| `!;` | Commits proof search at that point | It changes the search; justify discarded alternatives |

See TBoS fifth edition [§19.2, Left and Right Rules](https://shenlanguage.org/TBoS/242.html).

## Start with a concrete record

Use a synonym when only a second name is needed; it adds no datatype proof rules. Here the double
line lets the body recover the fields' types from a `book` hypothesis.

```shen
(synonyms year number)

(datatype book
  T : string; A : string; Y : year;
  =================================
  [T A Y] : book;)

(define title-line
  {book --> string}
  [T A Y] -> (make-string "~A (~A, ~A)" T A Y))

(if (= (title-line ["Structure and Interpretation" "Abelson" 1985])
       "Structure and Interpretation (Abelson, 1985)")
    true
    (simple-error "book rendering changed"))
```

Keep the counterexample focused on the claimed contract:

<!-- reject: bad-book-year -->
```shen
(define bad-book-year
  {--> book}
  -> ["Structure and Interpretation" "Abelson" "1985"])
```

## Choose the relevant reference

- [Data, refinements, and typed guards](references/refinements.md): component types, `verified`,
  same-file predicate order, operation symbols and `fn`, and when a grammar guard needs refinement.
- [Proof search](references/proof-search.md): modes and genuine union types, tagged sums, cuts,
  annotations, and reducing the theory while diagnosing loops.
- [Indices, abstraction, and composition](references/invariants-and-abstraction.md): nonempty
  sequences, limits of automatic dependent proofs, abstract storage, and composing trusted operations.
- [Closures and state](../shen-thinking/references/closures-and-state.md): distinguish captured
  values, mutation, and persistent versions before choosing a type interface.
- [Typed grammar design](../shen-logic-grammars/references/typed-grammars.md): larger parsers,
  semantic actions, token types, and the parser's input/output contract.

Each reference is a separate runnable sequence; its blocks build on earlier accepted blocks in that
file. Blocks marked `reject: NAME` must fail at the named definition. An `untyped` block identifies
an intentional implementation boundary. Run `bifrost skills check --dir skills shen-types` to validate
these conventions and examples.

Use typed `head`/`tail` rather than untyped `hd`/`tl`. Load `defprolog` through an explicit untyped
boundary and give its public operations reviewed types; see
[`shen-logic-grammars`](../shen-logic-grammars/SKILL.md). For a heterogeneous collection, define a
common element type instead of assuming every element must have the same primitive representation.

## Sources

Use [*The Book of Shen*, fifth edition](https://shenlanguage.org/TBoS/index.html), especially
chapters 19–22, for the type rules and their operational interpretation. The downloadable program
filenames retain some older section labels; follow the linked section title when comparing editions.
Consult the pinned kernel's `types.shen`, `sequent.shen`, and `t-star.shen` for version-specific behavior.
