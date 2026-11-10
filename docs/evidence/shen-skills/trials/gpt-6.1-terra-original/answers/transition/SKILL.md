# Configured transition

A configured transition closes over its fee; balances are explicit values. Results are tagged to distinguish insufficient funds from a numeric balance. This version models nonnegative amounts and fees as a caller obligation.

```shen
(datatype action
  N : number; ========== [credit N] : action;
  N : number; ========== [debit N] : action;)
(datatype outcome
  N : number; ========== [balance N] : outcome;
  ______ insufficient : outcome;)
(define account-step {number --> number --> action --> outcome}
  Fee Balance [credit N] -> [balance (+ Balance N)]
  Fee Balance [debit N] -> insufficient where (< Balance (+ N Fee))
  Fee Balance [debit N] -> [balance (- Balance (+ N Fee))])
(define configure {number --> (number --> action --> outcome)}
  Fee -> (/. Balance (/. Act (account-step Fee Balance Act))))
(define check-t {boolean --> boolean} true -> true false -> (error "transition assertion failed"))
(check-t (= ((configure 1) 10 [debit 4]) [balance 5]))
(check-t (= ((configure 2) 10 [debit 4]) [balance 4]))
(check-t (= ((configure 1) 10 [debit 4]) ((configure 1) 10 [debit 4])))
(check-t (= ((configure 1) 10 [credit 4]) [balance 14]))
(check-t (= ((configure 1) 4 [debit 4]) insufficient))
```

The immutable input value is not overwritten; the arrow type alone does not guarantee purity, determinism, or nonnegative amounts.

Separately, this deliberately untyped factory captures a private mutable vector. The only exposed operation increments and returns the counter. Aliases share the closure; independent factory calls allocate independent vectors. These are behavioral tests, not guarantees of an arrow type.

<!-- untyped -->
```shen
(define counter-factory
  -> (let V (address-> (absvector 1) 0 0)
       (/. Ignored (let N (+ 1 (<-address V 0))
                     (do (address-> V 0 N) N)))))
(define counter-assert true -> true false -> (error "counter assertion failed"))
(let A (counter-factory)
  (let Alias A
    (let B (counter-factory)
      (do (counter-assert (= (A go) 1))
          (counter-assert (= (Alias go) 2))
          (counter-assert (= (B go) 1))
          (counter-assert (= (A go) 3))))))
```
