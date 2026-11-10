# Expression parser

Typed semantic actions build a tree; tail functions fold operators left while term nesting gives multiplication precedence. `compile` requires complete consumption. The number guard bridge and if rule are trusted logical refinements; they depend on the runtime meaning of `number?`.

```shen
(datatype ptok
 N : number; __________ N : ptok;
 S : symbol; __________ S : ptok;)
(datatype past
 N : number; __________ N : past;
 X : past; Y : past; ================== [add X Y] : past;
 X : past; Y : past; ================== [mul X Y] : past;)
(datatype pnumber
 _________________________________
 (number? X) : verified >> X : number;)
(datatype pif
 Test : boolean; Test : verified >> X : A; (not Test) : verified >> Y : A;
 ______________________________________________________________________
 (if Test X Y) : A;)
(defcc <pexpr> {(list ptok) ==> past}
 <pterm> <psum> := (<psum> <pterm>);)
(defcc <psum> {(list ptok) ==> (past --> past)}
 plus <pterm> <psum> := (/. L (<psum> [add L <pterm>]));
 <e> := (/. L L);)
(defcc <pterm> {(list ptok) ==> past}
 <pfactor> <pproduct> := (<pproduct> <pfactor>);)
(defcc <pproduct> {(list ptok) ==> (past --> past)}
 times <pfactor> <pproduct> := (/. L (<pproduct> [mul L <pfactor>]));
 <e> := (/. L L);)
(defcc <pfactor> {(list ptok) ==> past}
 N := N where (number? N);)
(define peval {past --> number}
 [add X Y] -> (+ (peval X) (peval Y))
 [mul X Y] -> (* (peval X) (peval Y))
 N -> N where (number? N)
 _ -> (simple-error "bad AST"))
(define pcheck {string --> boolean --> boolean}
 _ true -> true
 L false -> (simple-error L))
(define pfails {(list ptok) --> boolean}
 T -> (trap-error (do (compile (fn <pexpr>) T) false) (/. E true)))
(pcheck "precedence" (= (peval (compile (fn <pexpr>) [2 plus 3 times 4])) 14))
(pcheck "left association" (= (compile (fn <pexpr>) [1 plus 2 plus 3 times 4 times 5]) [add [add 1 2] [mul [mul 3 4] 5]]))
(pcheck "empty" (pfails []))
(pcheck "missing operand" (pfails [2 plus]))
(pcheck "trailing" (pfails [2 3]))
```

Types do not prove grammar completeness, termination, associativity, or the lexer. The tests exercise these cases only; they do not establish general correctness. Recursion consumes an operator and operand before repeating.
