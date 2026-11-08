# Data, refinements, and typed guards

Read in order, with `(tc +)` enabled before each load. Sections: [component types](#component-types),
[runtime refinements](#runtime-refinements), [symbols and functions](#symbols-and-functions), and
[grammar guards](#grammar-guards). The examples use their own names and follow the techniques in
TBoS [§19.6](https://shenlanguage.org/TBoS/249.html) and
[§21.4](https://shenlanguage.org/TBoS/282.html).

## Component types

A side condition can recognize the literal `kg`; it cannot use a constructor's pattern variable to
discover the value that variable will hold later. Give a restricted component its own type and make
that a premise of the enclosing rule. Literal patterns may still satisfy a side condition; the
problem is asking a syntax-time test to establish a fact about an arbitrary runtime value.

Use `measure-unit`, not Shen's existing `unit` type. TBoS
[§17.8, p.204](https://shenlanguage.org/TBoS/218.html) uses `unit` for unknown objects returned by
operations such as `read-file`; extending it would not establish this finite vocabulary.

```shen
(datatype measure-unit
  if (element? U [m s kg])
  ________________________
  U : measure-unit;)

(datatype qty
  X : number; U : measure-unit;
  =============================
  [X U] : qty;)

(define scale
  {number --> qty --> qty}
  K [N U] -> [(* K N) U])

(if (= (scale 3 [2 kg]) [6 kg])
    true
    (simple-error "quantity scaling changed"))
```

An unknown object is not evidence of membership in the measurement vocabulary:

<!-- reject: unknown-measure -->
```shen
(define unknown-measure
  {unit --> qty}
  U -> [1 U])
```

## Runtime refinements

A successful `where G` adds `G : verified` to the clause's hypotheses. The following rule trusts the
connection between `pos?` and the intended type. The literal introduction rule separately evaluates
the predicate on ground syntax. Keep the predicate stable and free of effects that could invalidate
that connection.

```shen
(define pos?
  {number --> boolean}
  N -> (and (integer? N) (> N 0)))

(datatype pos
  ________________________________
  (pos? N) : verified >> N : pos;

  if (number? N) if (pos? N)
  __________________________
  N : pos;

  N : pos;
  __________
  N : number;)

(define mk-pos
  {number --> pos}
  N -> N where (pos? N)
  N -> (error "~A is not a positive integer~%" N))

(define double-pos
  {pos --> pos}
  N -> (mk-pos (* 2 N)))

(if (= (double-pos 21) 42)
    true
    (simple-error "positive doubling changed"))
```

The earlier definition of `pos?` and its side condition can occur in the same file. Datatype expansion
and execution of the resulting side-condition code are different phases: the predicate must exist
when proof search calls it. Test the actual load order, including any reader-time macros. A separate
prerequisite file is useful when that order requires it; it is not universally required.

`mk-pos` promises a positive result when it returns normally; invalid input raises an error. The
checker does not prove termination or absence of exceptions. Arithmetic loses the refinement unless
a suitable rule re-establishes it:

<!-- reject: half-pos -->
```shen
(define half-pos
  {pos --> pos}
  N -> (/ N 2))
```

For a conjunctive guard, expose its two facts explicitly when a proof needs them:

```shen
(datatype verified-and
  P : verified, Q : verified >> R;
  ________________________________
  (and P Q) : verified >> R;)
```

Do not add an unconditional `N : pos` axiom to make `half-pos` pass. Audit every rule that connects
runtime predicates to types, including what happens if the predicate is redefined or consults mutable
state. Membership at one instant is not a lifetime guarantee for a mutable object.

## Symbols and functions

In data, `+` is a symbol; `(fn +)` obtains the function. The checker also has function typings for
named operations, so a direct constant pattern such as `[+ X Y]` can encounter a typing conflict.
Describe the permitted operation symbols with a component datatype, then give their function values
the intended application type. This is the pattern in TBoS
[§21.4](https://shenlanguage.org/TBoS/283.html), not a prohibition on operator symbols in data.

```shen
(datatype arithmetic-op
  if (element? Op [+ *])
  ______________________
  Op : arithmetic-op;

  Op : arithmetic-op;
  _______________________________________
  (fn Op) : (number --> number --> number);)

(datatype arithmetic-expr
  N : number;
  ____________________
  N : arithmetic-expr;

  Op : arithmetic-op; X : arithmetic-expr; Y : arithmetic-expr;
  ===========================================================
  [Op X Y] : arithmetic-expr;

  ______________________________________
  (number? X) : verified >> X : number;)

(define calculate-op
  {arithmetic-expr --> number}
  N -> N where (number? N)
  [Op X Y] -> ((fn Op) (calculate-op X) (calculate-op Y)))

(if (= (calculate-op [* [+ 2 3] 4]) 20)
    true
    (simple-error "operation-symbol evaluation changed"))
```

The application rule is a trusted connection to the named operations' contracts. Extending
`arithmetic-op` means checking that each added operation really has that contract.

## Grammar guards

Use TBoS [§22.2, Parsing Bytes to Numbers](https://shenlanguage.org/TBoS/293.html) for typed
Shen-YACC. A guard can simply select an already-typed result; it then needs no extra refinement rule:

```shen
(defcc <digit>
  {(list number) ==> number}
  N := N where (element? N [0 1 2 3 4 5 6 7 8 9]);)

(if (= (compile (fn <digit>) [3]) 3)
    true
    (simple-error "guarded numeric grammar changed"))
```

When an action's result type depends on the guard, its proof needs the guard's evidence. Shen-YACC
guards compile to conditionals. The default typing of `if` does not expose those branch assumptions;
this client therefore fails even though its true branch is guarded at runtime:

<!-- reject: unrefined-if -->
```shen
(define unrefined-if
  {number --> pos}
  N -> (if (pos? N) N 1))
```

The following additional rule exposes the facts needed by each branch. `if` still evaluates only
the selected branch.

```shen
(datatype refine-if
  Test : boolean; Test : verified >> X : A; (not Test) : verified >> Y : A;
  _________________________________________________________________________
  (if Test X Y) : A;)

(define pos-or-one
  {number --> pos}
  N -> (if (pos? N) N 1))

(if (and (= (pos-or-one -5) 1) (= (pos-or-one 7) 7))
    true
    (simple-error "conditional refinement changed"))

(datatype move
  if (element? D [north south east west])
  _______________________________________
  D : direction;

  ____________________________________________________________
  (element? D [north south east west]) : verified >> D : direction;

  D : direction; N : number;
  ==========================
  [go D N] : move;)

(defcc <move>
  {(list symbol) ==> move}
  go <dir> <times> := [go <dir> <times>];)

(defcc <dir>
  {(list symbol) ==> direction}
  D := D where (element? D [north south east west]);)

(defcc <times>
  {(list symbol) ==> number}
  twice := 2;
  <e> := 1;)

(if (= (compile (fn <move>) [go north twice]) [go north 2])
    true
    (simple-error "refined grammar result changed"))
```

Measure the extra proof search on the actual module. There is no universal percentage overhead.
`(preclude [refine-if])` disables this theory; `(include [refine-if])` enables it again. Keep it active
where its branch facts are needed. For a larger parser, read
[typed grammar design](../../shen-logic-grammars/references/typed-grammars.md).
