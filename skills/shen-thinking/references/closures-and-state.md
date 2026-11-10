# Closures, preparation and state

Use this reference when choosing what a function should retain, when preparation should run, or whether
state belongs inside a closure. The design connection to *Let Over Lambda* is a retained lexical
environment. Apply it with Shen's partial application, function syntax and state operations.

## Contents

- [Capture and partial application](#capture-and-partial-application)
- [Preparation placement](#preparation-placement)
- [Private mutable state](#private-mutable-state)
- [Explicit state with captured policy](#explicit-state-with-captured-policy)
- [Replay, observations and proof scope](#replay-observations-and-proof-scope)
- [Closures and macros](#closures-and-macros)
- [Sources](#sources)

## Capture and partial application

A closure can retain values after the expression that bound them has returned. Capture does not by
itself imply mutation. In this example the captured offset never changes:

```shen
(define closure-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(define make-adder
  {number --> (number --> number)}
  Offset -> (/. X (+ Offset X)))

(closure-check "captured offset"
  (= (let Add10 (make-adder 10) (Add10 7)) 17))

(closure-check "independent configurations"
  (= (let Add10 (make-adder 10)
          Add20 (make-adder 20)
       [(Add10 1) (Add20 1) (Add10 1)])
     [11 21 11]))

(closure-check "partial application"
  (= (map (+ 10) [1 2 3]) [11 12 13]))

(closure-check "named function as a value"
  (= (map (fn reverse) [[1 2] [3 4]]) [[2 1] [4 3]]))
```

Use the form that says what is fixed and what remains to be supplied:

| Expression | Meaning |
|---|---|
| `+` | Symbol data |
| `(fn +)` | The named addition function as a value |
| `(+ 10)` | A function with its first argument already supplied |
| `(/. X (+ 10 X))` | An explicit anonymous function |
| `(let Offset 10 (/. X (+ Offset X)))` | A function retaining a lexical binding |

Partial application often removes an unnecessary lambda. A lambda remains useful for reordering
arguments, combining operations or making a captured dependency clear. Shen's distinction between a
symbol and the callable value designated by `fn` matters for portability.

The ordinary function type connects the captured argument and later input:

<!-- reject: wrong-adder-input -->
```shen
(define wrong-adder-input
  {--> number}
  -> ((make-adder 10) tea))
```

## Preparation placement

Work before the returned lambda runs when the factory is called. Work in its body runs when the closure
is invoked. Here the input list is reduced to one offset before the reusable function is returned:

```shen
(define total-offset
  {(list number) --> number}
  [] -> 0
  [X | Xs] -> (+ X (total-offset Xs)))

(define prepare-adder
  {(list number) --> (number --> number)}
  Offsets -> (let Offset (total-offset Offsets)
              (/. X (+ Offset X))))

(closure-check "prepared sum reused"
  (= (let Add (prepare-adder [2 3 5]) [(Add 1) (Add 7)])
     [11 17]))
```

`(/. X (+ (total-offset Offsets) X))` would retain the list and recompute the total on each invocation.
For these immutable numeric inputs both yield the same answers. The placement decides which work is
done, when it is done, and what is retained. Do not infer a speedup without measuring the relevant port.

The timing difference is observable when the source is mutable. The prepared closure snapshots the
cell at construction; the other closure reads it at each invocation:

```shen
(define snapshot-adder
  {(vector number) --> (number --> number)}
  Cell -> (let Offset (<-vector Cell 1) (/. X (+ Offset X))))

(define fresh-adder
  {(vector number) --> (number --> number)}
  Cell -> (/. X (+ (<-vector Cell 1) X)))

(closure-check "construction versus invocation"
  (= (let Cell (vector 1)
          Init (vector-> Cell 1 2)
          Snapshot (snapshot-adder Cell)
          Fresh (fresh-adder Cell)
          Before (Fresh 1)
          Write (vector-> Cell 1 7)
       [(Snapshot 1) Before (Fresh 1)])
     [3 3 8]))
```

This is an observation of one deliberately mutable dependency, not a general performance claim.
Before moving preparation across a lambda, check whether it reads mutable state, performs I/O, allocates
storage callers should not share, or must observe fresh configuration. Capturing a vector retains access
to that vector; it does not copy or freeze it. A retained policy also does not freeze every global
function or external dependency the implementation may call.

## Private mutable state

Shen's `let` binds a value; it is not lexical assignment. Use an explicit mutable object when mutation
is intended. `set` and `value` refer to global variables, not to a local binding of the same spelling.

The following factory allocates a fresh one-element vector for each counter. Both operations return a
number. An unknown command produces an explicit runtime error; the `symbol` argument type does not
restrict the protocol to `inc` and `peek`.

```shen
(define counter-dispatch
  {(vector number) --> symbol --> number}
  Cell peek -> (<-vector Cell 1)
  Cell inc -> (let Next (+ (<-vector Cell 1) 1)
                  Write (vector-> Cell 1 Next)
               Next)
  _ _ -> (simple-error "unknown counter command"))

(define make-counter
  {number --> (symbol --> number)}
  Start -> (let Cell (vector 1)
               Init (vector-> Cell 1 Start)
             (counter-dispatch Cell)))

(closure-check "counter sequence"
  (= (let C (make-counter 0) [(C inc) (C inc) (C peek)])
     [1 2 2]))

(closure-check "aliases share a cell"
  (= (let C (make-counter 0)
          Alias C
          First (C inc)
       (Alias peek))
     1))

(closure-check "factories allocate independent cells"
  (= (let C1 (make-counter 0)
          C2 (make-counter 0)
          First (C1 inc)
       [(C1 peek) (C2 peek)])
     [1 0]))
```

The closure gives callers a restricted operation interface. Its arrow type does not describe mutation,
exclusivity, invocation order or the relationship between successive results. Introduce additional
judgments or an explicit state model if those are properties the application needs to enforce.

## Explicit state with captured policy

For replay, branching simulations or comparison, make the state an input and return the next state.
Stable policy can still be captured. Here `Step` is fixed while `State` is supplied for every command;
the pair contains `(next-state * observed-result)`:

```shen
(define counter-step-by
  {number --> number --> symbol --> (number * number)}
  Step State inc -> (let Next (+ State Step) (@p Next Next))
  _ State peek -> (@p State State)
  _ _ _ -> (simple-error "unknown counter command"))

(define configured-counter-step
  {number --> (number --> symbol --> (number * number))}
  Step -> (counter-step-by Step))

(closure-check "same explicit input replays"
  (= (let Decide (configured-counter-step 3)
       [(Decide 0 inc) (Decide 0 inc)])
     [(@p 3 3) (@p 3 3)]))

(closure-check "caller carries the next state"
  (= (let Decide (configured-counter-step 3)
          First (Decide 0 inc)
       (Decide (fst First) inc))
     (@p 6 6)))

(closure-check "peek leaves explicit state unchanged"
  (= ((configured-counter-step 3) 9 peek) (@p 9 9)))
```

| Design | Caller supplies | Changing state |
|---|---|---|
| Mutable counter closure | Command | Shared cell behind the closure |
| Explicit transition | Policy, state and command | Input and returned value |
| Configured transition closure | State and command | Explicit; selected policy is retained |

These definitions use only their inputs and pure arithmetic for valid commands, so their replay
behavior can be reasoned about from the bodies. The arrow signatures alone do not establish purity.

## Replay, observations and proof scope

For a richer domain, a useful conceptual interface is:

```text
transition : Policy --> State --> Command --> (State * (list Effect))
```

Here an `Effect` is data describing an intended action. A separate interpreter performs it. Applying
`transition` to policy can create a configured decision function while keeping state and effects
inspectable. Define the domain types and effect interpreter before treating this sketch as runnable code.

For repeatable decisions, specify the source of time, randomness, configuration and external reads.
Persist the required inputs, policy version and effect descriptions. A closure is a useful callable
view, but serializing or comparing a policy generally calls for an explicit data representation and a
specified correspondence between that data and the derived operation.

Distinguish these obligations:

| Obligation | Evidence to seek |
|---|---|
| Legal client uses | Typechecking against justified interface judgments, with invalid clients rejected |
| Operations satisfy their laws | Reductions, invariant arguments, tests or a formal implementation proof |
| Intended execution | The port/runtime and semantics on which the reasoning depends |

A persistent stack supplies a concrete next step: a closure can represent an immutable value whose
operations satisfy equations. See [Persistent stacks and indexed interfaces](indexed-stacks.md) for
two implementations, typed clients and the explicit trust boundary.

## Closures and macros

Start with a function when composing runtime values. Use a macro when the task needs syntax processing,
new bindings or changed evaluation structure. Review introduced variable names, evaluation count,
package qualification and interactions between macro expansions. Common Lisp macro examples cannot be
transcribed without considering Shen's fixed-point expansion model.

The [macro reference](../../shen-logic-grammars/references/macros-and-loading.md) covers helper loading
and these phase distinctions.

## Sources

- Mark Tarver, [*The Book of Shen*](https://shenlanguage.org/TBoS/index.html), chapters 6–8 on
  higher-order programming, assignment and vectors; chapter 16 on methodology; chapter 21 on abstract
  operations and their representations.
- Doug Hoyte, [*Let Over Lambda*, chapter 2](https://letoverlambda.com/index.cl/guest/chap2.html), on
  closure environments and let/lambda placement. The examples here are Shen adaptations.
- Shen specification: [symbol semantics and `fn`](https://shenlanguage.org/SD/SymbolSemantics.html),
  [vectors](https://shenlanguage.org/SD/Vectors.html),
  [global variables](https://shenlanguage.org/SD/External.html) and
  [macros](https://shenlanguage.org/SD/Macros.html).
