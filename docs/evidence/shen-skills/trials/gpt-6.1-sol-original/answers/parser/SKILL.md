# Typed expression grammar

Two grammar levels give multiplication precedence. Each level parses a head
and a list of following operands, then folds left with checked semantic actions.
The token union admits heterogeneous number/operator lists without pretending
they are homogeneous symbol lists. No parse-result declaration is trusted.

```shen
(datatype p.token
  N : number;
  ______________
  N : p.token;

  if (element? S [plus times])
  __________________________
  S : p.token;)

(datatype p.expr
  N : number;
  ______________
  N : p.expr;

  L : p.expr; R : p.expr;
  ===============================
  [p-add L R] : p.expr;

  L : p.expr; R : p.expr;
  ===============================
  [p-mul L R] : p.expr;)

(datatype p.number-refinement
  ______________________________________
  (number? X) : verified >> X : number;)

(datatype p.refine-if
  Test : boolean; Test : verified >> X : A; (not Test) : verified >> Y : A;
  ______________________________________________________________________
  (if Test X Y) : A;)

(define p.adds
  {p.expr --> (list p.expr) --> p.expr}
  Acc [] -> Acc
  Acc [X | Xs] -> (p.adds [p-add Acc X] Xs))

(define p.muls
  {p.expr --> (list p.expr) --> p.expr}
  Acc [] -> Acc
  Acc [X | Xs] -> (p.muls [p-mul Acc X] Xs))

(defcc <p-expr>
  {(list p.token) ==> p.expr}
  <p-term> <p-plus-tail> := (p.adds <p-term> <p-plus-tail>);)

(defcc <p-plus-tail>
  {(list p.token) ==> (list p.expr)}
  plus <p-term> <p-plus-tail> := [<p-term> | <p-plus-tail>];
  <e> := [];)

(defcc <p-term>
  {(list p.token) ==> p.expr}
  <p-factor> <p-times-tail> := (p.muls <p-factor> <p-times-tail>);)

(defcc <p-times-tail>
  {(list p.token) ==> (list p.expr)}
  times <p-factor> <p-times-tail> := [<p-factor> | <p-times-tail>];
  <e> := [];)

(defcc <p-factor>
  {(list p.token) ==> p.expr}
  N := N where (number? N);)

(define p.parse
  {(list p.token) --> p.expr}
  Tokens -> (compile (fn <p-expr>) Tokens))

(define p.eval
  {p.expr --> number}
  [p-add L R] -> (+ (p.eval L) (p.eval R))
  [p-mul L R] -> (* (p.eval L) (p.eval R))
  N -> N where (number? N)
  _ -> (error "invalid expression"))

(define p.rejects?
  {(list p.token) --> boolean}
  Tokens -> (trap-error (do (p.parse Tokens) false) (/. E true)))

(define p.assert
  {boolean --> boolean}
  true -> true
  false -> (error "parser assertion failed"))

(p.assert (= (p.parse [2 plus 3 times 4]) [p-add 2 [p-mul 3 4]]))
(p.assert (= (p.eval (p.parse [2 plus 3 times 4])) 14))
(p.assert (= (p.parse [1 plus 2 plus 3]) [p-add [p-add 1 2] 3]))
(p.assert (= (p.parse [2 times 3 times 4]) [p-mul [p-mul 2 3] 4]))
(p.assert (= (p.eval (p.parse [1 plus 2 plus 3 times 4 times 5])) 63))
(p.assert (p.rejects? []))
(p.assert (p.rejects? [plus 2]))
(p.assert (p.rejects? [2 plus]))
(p.assert (p.rejects? [2 times plus 3]))
(p.assert (p.rejects? [2 3]))
(p.assert (p.rejects? [2 plus 3 4]))
```

`compile` is used at the public boundary, rather than invoking a nonterminal
and ignoring its remainder: it requires complete consumption. Trailing valid
tokens are tested explicitly. Failure raises an error, which `p.rejects?` catches
only in the negative tests. Input is already tokenized; unknown token symbols
are outside the typed input domain. There are no parentheses or unary operators.

The checked grammar actions and folds establish AST shape; the `number?`
verified rule is the standard guard refinement and depends on that predicate's
meaning. The general `if` rule propagates verified guards from generated grammar
code. These are explicit type-system extensions, not untyped parse-result axioms.
Types alone do not prove precedence, associativity, complete consumption,
unambiguity, parser termination, or arithmetic correctness. AST-shape tests matter
because evaluation of associative operators alone cannot detect grouping errors.
