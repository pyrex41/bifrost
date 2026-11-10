# Explicit account transition and private counter

The fee is captured at configuration time, while balance and action are explicit inputs. The example assumes nonnegative balances, amounts, and fees; these constraints are not enforced by the types. `insufficient` is an explicit result and no balance is mutated.

```shen
(datatype action
 N : number; ================= [credit N] : action;
 N : number; ================= [debit N] : action;)
(datatype outcome
 N : number; ================= [balance N] : outcome;
 _____________________________ insufficient : outcome;)
(define account-step {number --> number --> action --> outcome}
 _ B [credit N] -> [balance (+ B N)]
 Fee B [debit N] -> insufficient where (< B (+ N Fee))
 Fee B [debit N] -> [balance (- B (+ N Fee))])
(define account {number --> (number --> action --> outcome)}
 Fee -> (account-step Fee))
(define tcheck {string --> boolean --> boolean}
 _ true -> true
 L false -> (simple-error L))
(tcheck "configuration and replay"
 (= (let A (account 1) B (account 3) Old 10
      [(A Old [debit 4]) (B Old [debit 4]) (A Old [debit 4]) [balance Old] (A Old [credit 2]) (A Old [debit 10])])
    [[balance 5] [balance 3] [balance 5] [balance 10] [balance 12] insufficient]))

(define ctr-dispatch {(vector number) --> symbol --> number}
 Cell peek -> (<-vector Cell 1)
 Cell inc -> (let N (+ (<-vector Cell 1) 1) W (vector-> Cell 1 N) N)
 _ _ -> (simple-error "unknown counter command"))
(define counter {number --> (symbol --> number)}
 Start -> (let Cell (vector 1) W (vector-> Cell 1 Start) (ctr-dispatch Cell)))
(tcheck "alias and separate factories"
 (= (let C (counter 0) Alias C D (counter 0) First (C inc)
       [(Alias peek) (D peek) (Alias inc) (C peek)])
    [1 0 2 2]))
```

The counter owns a fresh vector per factory invocation; its aliases share it. This is behavioral reasoning about these bodies and runtime allocation, not an arrow-type guarantee of freshness, purity, replay or command protocol. The transition similarly relies on the given arithmetic and immutable numeric policy, not a type-level effect system.
