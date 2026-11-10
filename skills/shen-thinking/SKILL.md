---
name: shen-thinking
description: Design and change Shen programs using functional decomposition, closures, sequent-calculus types, Prolog, typed grammars and macros. Use before writing Shen (.shen files or REPL work), when translating rules or constraints into code, or when choosing representations, state boundaries and proof obligations.
---

# Thinking in Shen

Start with the problem's operations, laws and changing inputs. Shen supplies ordinary pattern-directed
functions and higher-order programming alongside extensible types, Prolog and typed grammars. Choose
the combination that makes the specification explicit and the implementation easy to check.

The failure this skill guards against: a generator contains a table of finished answers but none of the
rules that produced them. It passes examples while providing no account of correctness or extension.
Keep the rules or source facts in the program. A derived table is useful when its derivation, refresh
policy and consistency checks are explicit; a table can also be the specification itself.

## Workflow

For a substantial recipe, record **problem → judgment → assumptions → rules → implementation →
evidence → limitations**. A judgment states exactly what the active rules should establish; the
assumptions include data shape, effects, trusted declarations and runtime. Tests and typechecking
answer different questions, and neither alone proves termination or all intended laws.

1. **State the operations and laws.** What are the inputs, outputs, legal operations and expected
   equations? Separate output validity from completeness, uniqueness, termination and resource bounds.
   For a stack, start with `top(push(x, s)) = x` and `pop(push(x, s)) = s`.
2. **Separate fixed context from changing state.** What can a closure retain? What must remain an
   explicit input and result for replay, inspection or independent callers? Decide whether each
   preparation step happens once per factory call or on every invocation.
3. **Choose the mechanism by the work it must do.**

   | Work | Start with | Read next |
   |---|---|---|
   | Decompose cases or transform a value | `define`, patterns, `where`, small helpers | Example below |
   | Reuse a computation with selected arguments fixed | Partial application, `/.`, `let` | [Closures and state](references/closures-and-state.md) |
   | Express a value invariant or an admissible operation | `datatype` sequents, checked functions | `shen-types` |
   | Search relations or enumerate solutions | `defprolog`, explicit query modes | `shen-logic-grammars` |
   | Recognize tokens and compute their meaning | Typed `defcc` | `shen-logic-grammars` |
   | Execute rules supplied as values | A typed interpreter over the rule data | `shen-logic-grammars` |
   | Change syntax, binding or evaluation structure | `defmacro` | `shen-logic-grammars` |
   | Delay work or consume a sequence incrementally | `freeze` / `thaw`, with a demand contract | TBoS §17.4 and §24.6 |

   An explicit worklist or visited set can express a necessary search strategy. Prolog also needs an
   operational account of cycles, goal order and termination. Dynamic predicates may suit changing
   relations; custom rule data is appropriate when the program must inspect, store or version them.
4. **Choose a representation that supports the required observations.** Trees, lists and symbols make
   structure accessible; strings are appropriate for text. Tags can distinguish alternatives or carry
   domain meaning. Closures can implement callbacks, configured operations and persistent values.
   If values must be serialized or compared semantically, define the data representation or observation
   operations explicitly. Generic printing or equality is not a serialization or equivalence contract.
5. **State useful invariants and their preservation.** Give ordinary functions signatures. Use concrete
   datatype rules to describe shapes and refinements, including internal invariants that transformations
   must preserve. A checked input boundary is valuable, but downstream preservation can be equally
   valuable. Keep changing vocabulary in data unless a fixed vocabulary is part of the intended type.
6. **Identify the trusted steps.** A `verified` bridge, `declare`, or operation axiom is a claim that
   needs justification. Client checking under those rules does not establish an untyped implementation's
   laws. Ordinary arrow types do not establish purity, termination, effect freedom or a temporal protocol.
7. **Check the actual claims.** Turn checking on before loading typed code. Assert predicted values and
   laws. Isolate deliberately invalid clients and verify the intended type error. Exercise extensions,
   empty inputs, aliasing and relevant termination limits. Review with `shen-review`.

## Find the common computation

Before introducing an interpreter or a new type, see whether the repeated work is already a small
higher-order function. Here the stopping rule and step vary, while the recursive computation stays fixed:

```shen
(define repeat-until
  {(A --> boolean) --> (A --> A) --> A --> A}
  Done? _ X -> X where (Done? X)
  Done? Step X -> (repeat-until Done? Step (Step X)))

(define reach
  {number --> number}
  Target -> (repeat-until (/. X (>= X Target)) (+ 3) 0))

(if (= (reach 10) 12)
    true
    (simple-error "repeat-until: expected first value at or above ten"))
```

`(+ 3)` is a partial application. The lambda captures `Target`. The signature connects the predicate,
step and state types; it does not prove that an arbitrary step eventually satisfies the predicate.
For the example, the monotonically increasing sequence starting at zero reaches a value at least ten.

This client breaks the shared state type, so it must fail before execution:

<!-- reject: wrong-repeat -->
```shen
(define wrong-repeat
  {--> number}
  -> (repeat-until (/. X (> X 10)) (/. X [X]) 0))
```

When adapting this pattern, say what establishes termination or supply a budget and an explicit exhausted
result. Do not mistake a well-typed infinite computation for a total one.

## From closures to logical interfaces

Read [Closures and state](references/closures-and-state.md) when choosing between captured policy,
private mutable storage and explicit transitions. It covers partial application, preparation placement,
aliasing, replay and the limits of an ordinary function type.

Read [Persistent stacks and indexed interfaces](references/indexed-stacks.md) when an abstract value can
be implemented by a closure or when legal operations depend on a structural index. It works through
operation laws, two representations, accepted clients, rejected empty-stack operations and the boundary
between asserting an interface and proving its implementation.

Use both levels deliberately: a closure supplies an implementation; Shen's judgments can express which
uses are admissible. Write down the correspondence that connects the two.

## Keep one authoritative specification

When one concept has several executable views, identify the source and how the other views stay in sync.
For example, a sentence generator and a type-level lexicon must not silently maintain separate word
lists. Derive them from one source, or document and check the correspondence. Add a new word and verify
both acceptance and enumeration.

Likewise, `(list sentence)` establishes the type of each returned element under the active rules. An
empty list has that type too. Claims such as "all sentences exactly once" need a separate enumeration
argument and checks. A count alone can hide both a missing answer and a duplicate.

## Checking your code

Typecheck a file with the checker already enabled:

```bash
bifrost eval -e '(do (tc +) (load "prog.shen"))'
```

`load` takes the checking setting when it starts. Putting `(tc +)` inside the file does not retroactively
typecheck that load. Running code as a script is also insufficient evidence of checking. For a deliberate
untyped representation or Prolog layer, load it explicitly under `(tc -)`, then restore `(tc +)` before
loading its typed interface and clients. Report these boundaries honestly.

Run `bifrost skills check --dir skills --impl PORT` when changing these examples. The checker visits
linked Markdown references inside each skill too. Results asserted by the examples are runtime checks;
they are separate from the checker accepting their types.

## Reading

- Mark Tarver, [*The Book of Shen*, online fifth edition](https://shenlanguage.org/TBoS/index.html):
  chapters 3–8 on recursion, data and higher-order programming; chapter 16 on methodology; chapters 17–22 on types, proof
  control, abstraction and typed grammars. Check the displayed heading when following older program
  filenames, since a filename alone is not a reliable edition/section citation.
- Doug Hoyte, [*Let Over Lambda*, chapter 2](https://letoverlambda.com/index.cl/guest/chap2.html):
  lexical environments and closure construction. Apply the design idea using Shen's semantics;
  Common Lisp assignment and macro conventions require separate justification in Shen.
- `shen-types` for inference rules and proof search; `shen-logic-grammars` for relational and grammar
  execution; `shen-review` for the final review.
