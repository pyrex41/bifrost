---
name: shen-logic-grammars
description: Using Shen's built-in rule engines instead of hand-written loops — defprolog for relations, constraints and search; defcc (Shen-YACC) for parsing token sequences into meanings; a ruleset value plus a small typed interpreter for generative or changing rules; defmacro for notation. Covers how to give untyped Prolog a typed boundary. Use when a Shen task involves search, inference, relations, parsing, generation from rules, or a mini-language.
---

# Rule engines Shen already ships

When a problem's rules are a relation, a grammar or a little language, Shen already has the interpreter.
Writing your own search loop, tokenizer or rule matcher over your own format duplicates it, and hides the
rules inside control flow. This skill shows each engine, and how each joins typed code. Read
`shen-thinking` first for when to pick which.

## Prolog: relations, constraints, search

Use `defprolog` when the rules say who is related to whom, or "which X satisfy...", or need
backtracking. The clauses read like the domain's own rules: here, a course catalogue.

<!-- untyped -->
```shen
(defprolog requires          \\ Course requires Course
  cs201 cs101 <--;
  cs220 cs101 <--;
  cs301 cs201 <--;
  cs301 cs220 <--;
  cs350 cs301 <--;
  cs350 math210 <--;)

(defprolog completed         \\ Student has passed Course
  ada cs101 <--; ada cs201 <--; ada math210 <--;
  bob cs101 <--;)

(defprolog course            \\ derived, so there is no second list to keep in step
  C <-- (requires C _);
  C <-- (requires _ C);)

(defprolog needs             \\ transitive closure: two clauses
  C P <-- (requires C P);
  C P <-- (requires C Q) (needs Q P);)

(defprolog neg               \\ negation as failure (TBoS 25.6)
  P <-- (call P) ! (when false);
  _ <--;)

(defprolog missing
  S C <-- (requires C P) (neg (completed S P));)

(defprolog can-take
  S C <-- (course C) (neg (completed S C)) (neg (missing S C));)

(define prerequisites C -> (prolog? (findall P (needs (receive C) P) Ps) (return Ps)))
(define eligible S      -> (prolog? (findall C (can-take (receive S) C) Cs) (return Cs)))

(declare prerequisites [symbol --> [list symbol]])
(declare eligible      [symbol --> [list symbol]])
```

`defprolog` has no typed form: it expands to a `define` with no signature, and under `(tc +)` it fails
with `missing { in F`. So load the Prolog untyped (`(tc -)`, load, `(tc +)`) and give its entry points
types with `declare`. That `declare` is a **trusted** assertion, not a proof: keep the boundary small,
together and commented. Everything that uses it is then typechecked:

```shen
(define nub
  {(list A) --> (list A)}
  [] -> []
  [X | Xs] -> (nub Xs) where (element? X Xs)
  [X | Xs] -> [X | (nub Xs)])

(define report
  {symbol --> string}
  S -> (make-string "~A may take ~A" S (nub (eligible S))))

(nub (prerequisites cs350))
(report ada)
```

`(report ada)` is `"ada may take [cs220]"`. Adding a rule such as "or the requirement is waived" is one
more clause. In the hand-written version (an adjacency list, a worklist, a visited set and a copy of the
course list) it is an edit inside a loop, and "transitive prerequisite" is never stated anywhere.

Prolog notes:
- `receive` passes a Shen value into a query, `return` passes one out, `findall` collects all solutions,
  `!` cuts, and `(when P)` tests a Shen expression.
- Facts stay as facts. Don't build an association list and search it.
- The riddle in TBoS 27.19 is a model of stating constraints and letting search do the rest.

## Grammars: `defcc` (Shen-YACC)

Use `defcc` when the rules say which token sequences are allowed and what they mean. Each nonterminal is a
rule, alternatives are separated by `;`, and `:=` gives the meaning built from the parts. The meaning is
*structure*, never strings: a term that other rules can work on. Start with the term language, as a
datatype:

```shen
(datatype expr
  X : number;
  ___________
  X : expr;

  X : expr; Y : expr;
  ===================
  [add X Y] : expr;

  X : expr; Y : expr;
  ===================
  [sub X Y] : expr;

  X : expr; Y : expr;
  ===================
  [mul X Y] : expr;)

(datatype refine-number
  ______________________________________
  (number? X) : verified >> X : number;)
```

The tags are `add`, `sub` and `mul`, not `+`, `-` and `*`: in typed code a symbol that names a function
has that function's type, so `[+ X Y]` as a pattern fails to typecheck.

The grammar is untyped here, so it loads under `(tc -)`. Its entry point checks what it built, and that
check is what the `declare` relies on:

<!-- untyped -->
```shen
(defcc <expr>
  <term> + <expr> := [add <term> <expr>];
  <term> - <expr> := [sub <term> <expr>];
  <term>;)

(defcc <term>
  <factor> * <term> := [mul <factor> <term>];
  <factor>;)

(defcc <factor>
  N := N  where (number? N);
  [<expr>] := <expr>;)

(define expr?
  N -> true where (number? N)
  [Op X Y] -> (and (element? Op [add sub mul]) (expr? X) (expr? Y))
  _ -> false)

(define read-expr
  S -> (let E (compile (fn <expr>) (read-from-string S))
         (if (expr? E) E (error "not an expression: ~A~%" S))))

(declare read-expr [string --> expr])
```

`read-from-string` turns `"2 * (3 + 4)"` into the forms `[2 * [3 + 4]]`, and `[<expr>]` matches the
nested list. Everything after the boundary is typed:

```shen
(define calc
  {expr --> number}
  [add X Y] -> (+ (calc X) (calc Y))
  [sub X Y] -> (- (calc X) (calc Y))
  [mul X Y] -> (* (calc X) (calc Y))
  N -> N  where (number? N)
  _ -> (error "not an expression~%"))

(read-expr "2 * (3 + 4)")
(calc (read-expr "2 * (3 + 4) - 1"))
```

That gives `[mul 2 [add 3 4]]` and `13`. (The grammar is right-recursive, so `8 - 2 - 1` groups as
`8 - (2 - 1)`. A real one would collect the terms and fold them to the left.) The input arrives as a
string for a reason too: a typed list literal holds one element type, so a mixed token list like
`[2 * [3 + 4]]` can't even be written in typed code.

Typed grammars exist too: `{(list symbol) ==> move}` gives a `defcc` a type, and the checker proves that
it only produces well-formed results. See `shen-types` recipe 10, and TBoS 20.4 (English into typed logic)
and 22.2 (character codes into numbers).

## Rules as data, and a small interpreter

Use this when the rules generate or transform things, and they change, are many, or arrive at run time.
The ruleset is a plain value that can be printed, diffed, checked and stored. The interpreter is a few
short functions that never mention a domain word. This is the cure for the hard-coded table of strings:

```shen
(datatype item               \\ words are strings, references to rules are symbols
  X : string;
  ___________
  X : item;

  X : symbol;
  ___________
  X : item;)

(datatype rule
  N : symbol; As : (list (list item));
  ====================================
  [N | As] : rule;)

(datatype refine-string
  ______________________________________
  (string? X) : verified >> X : string;)

(define alternatives
  {item --> (list rule) --> (list (list item))}
  _ [] -> []
  N [[N | As] | _] -> As
  N [_ | G] -> (alternatives N G))

(define expand
  {(list rule) --> item --> number --> (list string)}
  _ X _ -> [X] where (string? X)
  _ _ 0 -> []
  G N D -> (mapcan (/. Alt (expand-all G Alt (- D 1))) (alternatives N G)))

(define expand-all
  {(list rule) --> (list item) --> number --> (list string)}
  _ [] _ -> [""]
  G [X | Xs] D -> (cross (expand G X D) (expand-all G Xs D)))

(define cross
  {(list string) --> (list string) --> (list string)}
  Hs Ts -> (mapcan (/. H (map (/. T (join H T)) Ts)) Hs))

(define join
  {string --> string --> string}
  H "" -> H
  H T -> (@s H " " T))

\\ The ruleset: change the language here, not the program.
(define orders
  {--> (list rule)}
  -> [[order [qty drink] [qty size drink]]
      [qty   ["one"] ["two"]]
      [size  ["small"] ["large"]]
      [drink ["tea"] ["coffee"] [drink "with" extra]]
      [extra ["milk"] ["lemon"]]])

(length (expand (orders) order 3))
(head (expand (orders) order 3))
```

It derives 36 orders, starting with `"one tea"`. Adding `["medium"]` to `size` gives 48 with no change to
the code. `drink` is recursive, and the depth bound keeps generation finite.

When the ruleset comes from outside (a file, a user), make "this ruleset is well formed" a boundary type:
a predicate such as `closed?` (every symbol it mentions has a rule), a rule
`(closed? G) : verified >> G : grammar;`, and one guarded function that is the only way to get a `grammar`.
Inside the boundary, use the plain shape types. This is the TBoS 21.4 and 28.3 pattern.

The same move scales to real languages: TBoS 24.3 is an SECD machine in a few typed clauses, and 25.7 is
Prolog itself written in typed Shen over Horn clauses as data. If a problem contains a language (formulas,
queries, commands), write the language: a datatype for its terms and one function per rule of evaluation.

## Macros: notation

Use `defmacro` when the *syntax* is repetitive and its meaning is fixed, never to hide data or where a
function would do. A macro rewrites forms as the file is read. Here, a notation for facts that writes the `defprolog` for you.
The macro calls a helper while it expands, so the helper must already be loaded, **from an earlier file**:
defining it below the macro in the same file fails with "undefined".

<!-- untyped -->
```shen
(define pairs->clauses
  [] -> []
  [A B | Ps] -> [A B <-- ; | (pairs->clauses Ps)])
```

<!-- untyped -->
```shen
(defmacro facts-macro
  [facts Rel | Pairs] -> [defprolog Rel | (pairs->clauses Pairs)])
```

<!-- untyped -->
```shen
(facts parent  tom bob  bob ann  bob cal)

(define children P -> (prolog? (findall C (parent (receive P) C) Cs) (return Cs)))

(children bob)
```

`(children bob)` gives `[cal ann]`. Also:
- Give a macro's pattern a distinctive head symbol. A pattern like `[X + Y]` also matches the `cons`
  forms the reader builds for list literals, and rewrites code you never meant it to.
- TBoS 21.5 is the model of a macro that pays: it reads ML-style datatype declarations and writes Shen
  sequent rules from them. When the notation is wrong for the problem, change the notation.

## Reading

*The Book of Shen*, 4th ed.: Prolog in chapters 25 to 27, Shen-YACC in 13 and 22, macros in 10, the SECD
machine in 24. Programs at <https://shenlanguage.org/TBoS/index.html>.
