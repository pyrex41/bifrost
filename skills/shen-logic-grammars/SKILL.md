---
name: shen-logic-grammars
description: Design Shen relations, parsers, rule interpreters and macros with explicit execution contracts. Use for Prolog queries and constraints, typed defcc grammars, generation from changing rules, syntax extensions, or typed boundaries around dynamic data and untyped engines.
---

# Relations, grammars and macros

Choose an engine by the required behavior, including its search order, failure policy and cost. State
the domain rules and the execution contract separately. For broader representation choices, including
closures and persistent data, read `shen-thinking`.

## Choose a route

| Problem | Start with | Contract to state | Runnable reference |
|---|---|---|---|
| Relations, alternative answers, constraints | `defprolog` | Ground inputs, goal/clause order, negation, cycles, duplicates, answer order | [Prolog search](references/prolog-search.md) |
| Token sequences and their meaning | Typed `defcc` | Token representation, consumption, precedence, associativity, backtracking, progress | [Typed grammars](references/typed-grammars.md) |
| Rules supplied as values or interpreted in several ways | Typed terms and an interpreter | Defined references, duplicate names, empty productions, recursion and work limits | [Bounded generation](references/bounded-generation.md) |
| New syntax, changed evaluation order, generated definitions | `defmacro` and packages | Expansion phases, scope, freshness, evaluation count, termination | [Macros and loading](references/macros-and-loading.md) |

Use ordinary pattern-matching functions, data structures, worklists or indexes when they fit the
required semantics or performance better. A catalog of objects can contain information independent of
a relation between those objects; deriving it from relation endpoints can silently lose isolated data.

## Make typed grammars the normal path

Give each nonterminal its input and semantic result type. This small parser needs no extra refinement
rules: its inputs and semantic actions already have the required numeric types.

```shen
(defcc <total>
  {(list number) ==> number}
  N <total> := (+ N <total>);
  <e> := 0;)

(if (= (compile (fn <total>) [1 2 3 4]) 10)
    passed
    (error "typed total parser returned the wrong value"))
```

The checker must reject an incompatible semantic result, for the intended reason:

<!-- reject: <wrong-total> -->
```shen
(defcc <wrong-total>
  {(list number) ==> number}
  N := wrong;)
```

Choose a result suitable for the consumer: a number, string, AST or semantic function. A list has one
element type, but that can be a union-like datatype covering numbers, symbols and nested lists. Mixed
tokens do not inherently require untyped code. The full reference builds a typed expression parser
whose semantic functions implement left associativity.

Shen-YACC guards compile to `if`. Add justified refinement rules only when a semantic result depends on
what a successful guard establishes; a guarded parser does not automatically need extra axioms. See
the guard/refinement recipe in `shen-types` and the [worked parser](references/typed-grammars.md).

## Separate checked code from trusted boundaries

- Load typed code with `(tc +)` enabled **before** `load`. Each reference's blocks load in order as
  separate files; this matters for expansion-time dependencies.
- Isolate `defprolog` and necessary untyped decoder/macro code. Document each `declare` as a trusted
  assertion; typechecking a caller does not check the declared implementation retroactively.
- Validate external shapes and domain conditions at the boundary. A predicate returning `boolean`
  does not by itself justify a rule promoting successful results into a refined type.
- Keep a parser typed after any dynamic decoding boundary. A generic Shen reader has Shen's syntax;
  use a suitable lexer when the requested language has different lexical rules.
- State the scope of a type claim. A semantic result type does not prove parser completeness,
  termination, exact enumeration, evaluation order or a particular complexity bound.

## Account for operational behavior

For Prolog, ground the relevant values before negation as failure. A negated query with free variables
does not enumerate a complement. `findall` needs the search to finish, cuts can remove answers, and
recursive closure over a cyclic graph needs an explicit termination strategy. The Prolog reference
includes a visited path and independent catalog facts.

For Shen-YACC, test complete input consumption and overlapping alternatives. Backtracking between
alternatives does not generally reconsider an earlier successful nonterminal inside an expansion.
Eliminate left recursion and ensure recursive cycles make progress. Check associativity with an
expression such as `8 - 2 - 1`, not only a successful single-operator parse.

For data-driven generation, define epsilon, unknown rules, duplicate names and output multiplicity.
Validate numeric depth budgets as nonnegative integers and ensure recursive work decreases them.
Finite depth can still produce an impractically large result set. The generator reference asserts
these boundary cases and keeps empty productions as empty token sequences until rendering.

For macros, load helpers before macros and macros before clients. Inspect the generated form, scope
syntax with packages, use fresh names for introduced bindings, and account for fixed-point expansion.
A distinctive trigger reduces accidental matches; it does not by itself prevent variable capture.

## Check behavior as well as types

Assert expected values and failures in executable examples; a printed answer is not a check. Include a
named `<!-- reject: function-name -->` block when claiming a particular definition is refused, so an
unrelated type error cannot satisfy the test. Check engine-specific adversarial cases: unbound negative
goals, cycles, isolated catalog entries, leftover tokens, malformed syntax, epsilon and invalid budgets.

Use `bifrost skills check --dir skills shen-logic-grammars` to run the entry point and references on the
active port. Apply `shen-review` before reporting the resulting program as complete.

## Sources

The [online fifth edition of *The Book of Shen*](https://shenlanguage.org/TBoS/index.html) develops
macros/packages in Chapter 10, parser execution in Chapter 13, typed grammars in Chapter 22, and
interpreters/Prolog in Chapters 24–27. Each reference links the relevant prose and programs. Cite
displayed section titles and exact URLs; a companion program's numbered filename alone does not
establish the prose section or edition.
