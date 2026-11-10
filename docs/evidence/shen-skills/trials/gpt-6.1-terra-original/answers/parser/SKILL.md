# Expression parser (incomplete)

The checked semantic domain and evaluator below are a partial solution. Tags keep the constructors disjoint; the recursive evaluator respects the tree it receives.

```shen
(datatype expression
  N : number; ========== [lit N] : expression;
  L : expression; R : expression; ========== [add L R] : expression;
  L : expression; R : expression; ========== [mul L R] : expression;)
(define evaluate {expression --> number}
  [lit N] -> N
  [add L R] -> (+ (evaluate L) (evaluate R))
  [mul L R] -> (* (evaluate L) (evaluate R)))
(define check-e {boolean --> boolean} true -> true false -> (error "expression assertion failed"))
(check-e (= (evaluate [add [lit 2] [mul [lit 3] [lit 4]]]) 14))
(check-e (= (evaluate [add [add [lit 1] [lit 2]] [lit 3]]) 6))
```

No parser is supplied. A complete implementation needs checked semantic actions constructing these trees, a left fold for each precedence level, and an entry point that refuses leftover tokens, with malformed-input tests. A trusted parse-result declaration alone would not establish those properties. These evaluator tests do not test parsing, associativity of parser output, or input consumption; typing alone would not prove termination or grammar correctness.
