---
name: shen-types
description: Cookbook for Shen's sequent-calculus type system — datatype rules, side conditions, double-line rules, left rules with >>, verified refinements and smart constructors, cut, synonyms, indexed types, abstract types, unions, typed defcc — plus the pitfalls that make the checker reject, loop or lie. Use when writing or debugging Shen datatypes or signatures, or when a Shen program should enforce a rule through its types.
---

# Shen's type system, used to the full

A type in Shen is **a set of objects defined by rules**. A `datatype` is not a record declaration bolted
onto a Lisp. It is a set of sequent-calculus inference rules, which Shen compiles into Prolog clauses that
the type checker runs. So when a problem arrives as rules (a grammar, a protocol, legal moves, valid
records), write the rules as types and let the checker prove the program obeys them.

Shen datatypes also differ from ML's algebraic types in two ways that matter:
- They **describe** data you already have (lists, symbols, strings, numbers). `[T A Y] : book` is a claim
  about an ordinary three-element list. There are no constructors to call and no wrapper to unwrap.
- They are **open**: a later, separate `datatype` can add rules to an existing type. So test with positive
  predicates (`string?`, `symbol?`) at run time, never with an `else` that assumes the type is closed.

Every example below has been typechecked under `(tc +)`. Blocks marked as rejected are refused by the
checker, which is the point of them.

## Anatomy of a rule

<!-- skip -->
```shen
(datatype name
  if <side condition>          \\ evaluated: a Shen expression run on the terms
  <premise>; <premise>;        \\ proved: X : T, or P : verified, or Hyps >> X : T
  ___________________          \\ single line: one rule, read downwards
  <conclusion>;

  <premises>;
  ===========                  \\ double line: a right rule AND a left rule
  <conclusion>;)
```

- **Side condition (`if`)**: evaluated on the term, so it decides things about literals and other ground
  syntax (19.1, 19.6 in *The Book of Shen*). It cannot decide things about a variable bound at run time.
- **Premise (`X : T`)**: proved by the checker, and works on variables.
- **Single line**: how to *build* a value of the type (a right rule), or a one-way fact.
- **Double line**: also gives the left rule: what a hypothesis of that shape tells the body. Pattern
  matching is typed by the right rules, but the body only learns the components' types through the left
  rule. Use `====` for constructors you will take apart.

## Cookbook

### 1. Enumeration (TBoS 19.1)

```shen
(datatype suit
  if (element? S [clubs diamonds hearts spades])
  ______________________________________________
  S : suit;)

(define red?
  {suit --> boolean}
  diamonds -> true
  hearts   -> true
  _        -> false)

(red? hearts)
```

Use an enumeration for a vocabulary that is fixed by the problem. If the words are domain data that
changes, keep them in data instead and type their shape (`string`, `symbol`).

### 2. Record by a double rule, with a synonym (TBoS 19.2)

```shen
(synonyms year number)

(datatype book
  T : string; A : string; Y : year;
  =================================
  [T A Y] : book;)

(define title-line
  {book --> string}
  [T A Y] -> (make-string "~A (~A, ~A)" T A Y))

(title-line ["Structure and Interpretation" "Abelson" 1985])
```

With `____` instead of `====` the same function is rejected, because the body knows `[T A Y]` is a whole
record and nothing about `T`:

<!-- reject -->
```shen
(datatype loose-book
  T : string; A : string; Y : year;
  _________________________________
  [T A Y] : loose-book;)

(define loose-title
  {loose-book --> string}
  [T A Y] -> (make-string "~A (~A, ~A)" T A Y))
```

A synonym names a type without adding rules, so it costs the checker nothing.

### 3. A component with a condition gets its own type (TBoS 19.4, 19.6)

A side condition on a constructor's *variable* makes patterns over that constructor untypable. Give the
component a type of its own and make it a premise:

```shen
(datatype unit
  if (element? U [m s kg])
  ________________________
  U : unit;)

(datatype qty
  X : number; U : unit;
  =====================
  [X U] : qty;)

(define scale
  {number --> qty --> qty}
  K [N U] -> [(* K N) U])

(scale 3 [2 kg])
```

### 4. Refinement by proof: `verified` and a smart constructor (TBoS 21.4, 28.3)

A `where G` guard adds `G : verified` to the hypotheses of its clause. A rule that turns that hypothesis
into a type is a refinement type. The guarded function is the only way in:

```shen
(define pos?
  {number --> boolean}
  N -> (and (integer? N) (> N 0)))
```

```shen
(datatype pos
  ________________________________
  (pos? N) : verified >> N : pos;

  if (integer? N) if (> N 0)
  __________________________
  N : pos;

  N : pos;
  __________
  N : number;)

(define mk-pos
  {number --> pos}
  N -> N  where (pos? N)
  N -> (error "~A is not positive~%" N))

(define double
  {pos --> pos}
  N -> (mk-pos (* 2 N)))

(double 21)
```

The literal rule lets `21` be a `pos`. Computed values become `pos` only through the guard. Arithmetic on a
`pos` gives a `number`, so the checker refuses a function that claims to keep the refinement without
checking it:

<!-- reject -->
```shen
(define half
  {pos --> pos}
  N -> (/ N 2))
```

`pos?` is in its own block above for a reason: a datatype is compiled when its file is *read*, before any
`define` in that file runs, so a side condition may only call kernel functions or functions loaded from an
*earlier* file. Here it is only a premise (`(pos? N) : verified`), which is fine, but if a side condition
called it, it would fail with "undefined function". The checker also trusts the predicate's *name*: if
you redefine `pos?`, `pos` silently means something else.

Split a conjunctive guard so `where (and P Q)` verifies both (from TBoS 21.4):

```shen
(datatype verified-and
  P : verified, Q : verified >> R;
  ________________________________
  (and P Q) : verified >> R;)
```

### 5. Cut in a rule (TBoS 20.3)

`!` among the premises commits to the first proof. Use it where alternatives are disjoint, so a failure
later doesn't send the checker through every remaining rule:

```shen
(datatype printable
  X : number; !;
  ______________
  X : printable;

  X : string; !;
  ______________
  X : printable;

  X : symbol;
  ______________
  X : printable;)

(define show-all
  {(list printable) --> string}
  [] -> ""
  [X | Xs] -> (@s (make-string "<~A>" X) (show-all Xs)))

(show-all [1 "two" three])
```

### 6. Length-indexed sequences (TBoS 20.8)

The index is a term the checker reasons about. `head` and `tail` become total:

```shen
(datatype seq
  ________________
  [] : (seq z A);

  X : A; Xs : (seq N A);
  ==========================
  [X | Xs] : (seq (s N) A);

  Xs : (list A) >> P;
  ____________________
  Xs : (seq N A) >> P;)

(define seq-head
  {(seq (s N) A) --> A}
  [X | _] -> X)

(define seq-tail
  {(seq (s N) A) --> (seq N A)}
  [_ | Xs] -> Xs)

(seq-head (seq-tail [a b c]))
```

Taking the head of something that may be empty is a type error:

<!-- reject -->
```shen
(define oops
  {--> symbol}
  -> (seq-head (seq-tail [a])))
```

The last rule *forgets* the index, so list functions still apply. A limit: signature variables are rigid,
so a clause cannot refine `N` to `z` or `(s M)` the way a GADT would. A function that recurses over a `seq`
of any length won't typecheck. State the index's shape in the signature, as above, and use the forgetting
rule for the rest.

### 7. Abstract type: operations declared, representation hidden (TBoS 21.1)

```shen
(define bag-of {A --> (list A)} X -> [X])
(define bag-add {A --> (list A) --> (list A)} X B -> [X | B])
(define bag-size {(list A) --> number} B -> (length B))

(datatype bag
  ___________________________
  bag-of : (A --> (bag A));

  _______________________________________
  bag-add : (A --> (bag A) --> (bag A));

  ________________________________
  bag-size : ((bag A) --> number);)

(define three-bag
  {--> (bag symbol)}
  -> (bag-add a (bag-add b (bag-of c))))

(bag-size (three-bag))
```

No rule says a `(bag A)` is a list, so client code that uses list operations on one is rejected:

<!-- reject -->
```shen
(define peek
  {(bag symbol) --> symbol}
  B -> (head B))
```

The operation axioms are trusted, not proved, and the implementing names still carry their list types. For
full hiding, follow 21.1: implement under `(tc -)` and publish only the axioms. Hide a representation only
when it must be free to change; opaque values can't be printed, compared or stored usefully.

### 8. Union with case analysis by explicit left rules (TBoS 20.7)

Write left rules with `>>` yourself when building and using a value differ:

```shen
(datatype either
  X : A;
  _______________________
  [left X] : (either A B);

  X : B;
  ________________________
  [right X] : (either A B);

  X : A >> P;
  ____________________________
  [left X] : (either A B) >> P;

  X : B >> P;
  _____________________________
  [right X] : (either A B) >> P;)

(define either->string
  {(either number string) --> string}
  [left N]  -> (str (+ N 1))
  [right S] -> S)

(either->string [left 41])
```

A left rule must make progress: it consumes its hypothesis. A rule that puts the same hypothesis back
loops until `maximum inferences exceeded`.

### 9. Making `if` refine as `where` does

Out of the box only `where` adds `Test : verified`. An `if` expression is an ordinary function. This rule
gives `if` the same power:

```shen
(datatype refine-if
  Test : boolean; Test : verified >> X : A; (not Test) : verified >> Y : A;
  _________________________________________________________________________
  (if Test X Y) : A;)

(define pos-or-one
  {number --> pos}
  N -> (if (pos? N) N 1))

(pos-or-one -5)
```

It costs about 5–7% more inferences everywhere it is in force. `(preclude [refine-if])` removes it, and
`(include [refine-if])` brings it back.

### 10. A typed grammar whose result is a datatype (TBoS 20.4, 22.2)

Guards in `defcc` compile to `if`, so a typed `defcc` with a guard needs rule 9 in force:

```shen
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
  D := D  where (element? D [north south east west]);)

(defcc <times>
  {(list symbol) ==> number}
  twice := 2;
  <e>   := 1;)

(compile (fn <move>) [go north twice])
```

## Pitfalls

1. **Circular rules.** `X : number ⊢ X : pos` together with `X : pos ⊢ X : number` loops. Point
   subtyping one way only.
2. **Partial side conditions.** Side conditions run during the search, on terms you never meant them for.
   `if (> N 0)` without a `number?` test first can raise an error while checking an unrelated function.
   Make them total.
3. **A side-condition function from the same file.** "undefined function": the datatype was compiled when
   the file was read. Use kernel functions, or load the predicate from an earlier file.
4. **A side condition on a constructor variable.** The pattern becomes untypable. Make it a component type
   (recipe 3).
5. **A single line where a double one was meant.** The body knows nothing about the components (recipe 2).
6. **Expecting GADT-style refinement.** Index the signature instead (recipe 6).
7. **Unsound by construction.** A rule with no premises and a free type, `___ X : A;`, proves everything.
   `verified` rules trust a predicate's name. Axioms over untyped code are unchecked. Keep these few,
   together, and commented as trust boundaries.
8. **Overlapping shapes.** A catch-all `[X Y]` beside `[op X]` multiplies backtracking. Make constructors
   disjoint by a leading tag, cut disjoint alternatives, and `preclude` datatypes a module doesn't use.
9. **Function names as tags.** In typed code a symbol that names a function has that function's type,
   so a pattern or rule over `[+ X Y]` fails. Tag with plain symbols (`[add X Y]`).
10. **Untyped kernel functions.** Some primitives have no type: use `head`/`tail` rather than `hd`/`tl`
   under `(tc +)`. A `define` named after a kernel function is refused ("not a legitimate function name").
11. **`defprolog` has no typed form.** Under `(tc +)` it fails with `missing { in F`. Load Prolog untyped
    and give its entry points types (see `shen-logic-grammars`).
12. **Heterogeneous literals.** A typed list holds one element type: `[1 two "three"]` is a type error
    unless a datatype such as recipe 5's `printable` covers all three.
13. **Mechanics.** `(tc +)` inside a file does not typecheck that file, nor does running it as a script;
    turn it on before `load`. Big,
    overlapping grammars cost inferences: watch the `typechecked in N inferences` line.

## Reading

*The Book of Shen*, 4th ed., chapters 19 to 28, and its programs: <https://shenlanguage.org/TBoS/index.html>.
The kernel's own `sources/types.shen`, `sequent.shen` and `t-star.shen` show what the checker actually does.
