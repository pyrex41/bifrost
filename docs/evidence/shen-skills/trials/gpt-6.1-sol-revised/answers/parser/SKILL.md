# Typed token parser and AST evaluator

Input is a flat token list, not source text. Numbers and symbols share a token type; only `plus` and `times` are recognized operators. The grammar constructs tagged AST nodes, not merely a trusted declared result. Tails are semantic functions accumulating on the left; product parsing is nested inside sum parsing.

```shen
(datatype e-token
  N : number;
  ____________
  N : e-token;

  S : symbol;
  ____________
  S : e-token;)

(datatype e-ast
  N : number;
  ==================
  [literal N] : e-ast;

  L : e-ast; R : e-ast;
  ====================
  [sum L R] : e-ast;

  L : e-ast; R : e-ast;
  ====================
  [product L R] : e-ast;)

(datatype e-number-guard
  ______________________________________
  (number? X) : verified >> X : number;)

(datatype e-branch-evidence
  Test : boolean; Test : verified >> X : A; (not Test) : verified >> Y : A;
  _____________________________________________________________________
  (if Test X Y) : A;)

(defcc <e-expression>
  {(list e-token) ==> e-ast}
  <e-term> <e-sums> := (<e-sums> <e-term>);)

(defcc <e-sums>
  {(list e-token) ==> (e-ast --> e-ast)}
  plus <e-term> <e-sums> := (/. Left (<e-sums> [sum Left <e-term>]));
  <e> := (/. Left Left);)

(defcc <e-term>
  {(list e-token) ==> e-ast}
  <e-number> <e-products> := (<e-products> <e-number>);)

(defcc <e-products>
  {(list e-token) ==> (e-ast --> e-ast)}
  times <e-number> <e-products> :=
    (/. Left (<e-products> [product Left <e-number>]));
  <e> := (/. Left Left);)

(defcc <e-number>
  {(list e-token) ==> e-ast}
  N := [literal N] where (number? N);)

(define e-evaluate
  {e-ast --> number}
  [literal N] -> N
  [sum L R] -> (+ (e-evaluate L) (e-evaluate R))
  [product L R] -> (* (e-evaluate L) (e-evaluate R)))

(define e-parse
  {(list e-token) --> e-ast}
  Tokens -> (compile (fn <e-expression>) Tokens))

(define e-fails?
  {(list e-token) --> boolean}
  Tokens -> (trap-error (do (e-parse Tokens) false) (/. E true)))

(define e-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(e-check "precedence AST"
  (= (e-parse [2 plus 3 times 4])
     [sum [literal 2] [product [literal 3] [literal 4]]]))
(e-check "precedence evaluation" (= (e-evaluate (e-parse [2 plus 3 times 4])) 14))
(e-check "repeated plus left AST"
  (= (e-parse [1 plus 2 plus 3])
     [sum [sum [literal 1] [literal 2]] [literal 3]]))
(e-check "repeated times left AST"
  (= (e-parse [2 times 3 times 4])
     [product [product [literal 2] [literal 3]] [literal 4]]))
(e-check "repeated mixed evaluation"
  (= (e-evaluate (e-parse [1 plus 2 times 3 times 4 plus 5])) 30))
(e-check "empty rejected" (e-fails? []))
(e-check "missing operand rejected" (e-fails? [2 plus]))
(e-check "adjacent operators rejected" (e-fails? [2 times plus 3]))
(e-check "leading operator rejected" (e-fails? [plus 2]))
(e-check "trailing number rejected" (e-fails? [2 plus 3 4]))
(e-check "trailing unknown symbol rejected" (e-fails? [2 plus 3 unknown]))
```

A malformed semantic action is rejected by the typechecker:

<!-- reject: <e-bad-action> -->
```shen
(defcc <e-bad-action>
  {(list e-token) ==> e-ast}
  N := [sum [literal 1]];)
```

`compile` requires complete consumption; the trailing-input assertions exercise that contract. There is no left recursion: each recursive tail alternative consumes an operator and an operand before recursion, while epsilon does not recurse. Thus the grammar terminates on finite input under the runtime's grammar semantics. Operator alternatives precede epsilon. AST equality, rather than arithmetic alone, distinguishes left from right associativity (addition/multiplication would hide the difference).

Trusted logical bridges: the `number?` verified rule assumes the built-in predicate really recognizes numbers; the conditional rule exposes facts valid in the selected branch. They are not parse-result declarations. Grammar semantic actions and evaluation bodies are checked under these rules. No untyped boundary or `declare` is used. Types alone do not prove exact language recognition, consumption, precedence, termination, purity, numeric precision, or correctness of a future lexer. Error tests recognize any parse-time exception, not a particular diagnostic; tests are finite evidence, not an exhaustive parser proof.
