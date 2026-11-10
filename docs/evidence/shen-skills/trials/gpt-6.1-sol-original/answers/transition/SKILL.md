# Configured transition and a separate private counter

An account configuration captures only a fee value. Balance is an explicit input;
there is no account cell. Credit has no fee; debit charges amount plus fee only
on success. Negative fees, balances, and action amounts raise domain errors.
Numbers use the runtime's arithmetic; this is not a decimal-money implementation.

```shen
(datatype t.action
  Amount : number;
  ==========================
  [credit Amount] : t.action;

  Amount : number;
  =========================
  [debit Amount] : t.action;)

(datatype t.result
  Balance : number;
  =========================
  [t-ok Balance] : t.result;

  ______________________________
  insufficient-funds : t.result;)

(define t.apply
  {number --> number --> t.action --> t.result}
  Fee Balance [credit Amount] -> [t-ok (+ Balance Amount)]
    where (and (>= Fee 0) (>= Balance 0) (>= Amount 0))
  Fee Balance [debit Amount] ->
    (let Cost (+ Amount Fee)
      (if (>= Balance Cost) [t-ok (- Balance Cost)] insufficient-funds))
    where (and (>= Fee 0) (>= Balance 0) (>= Amount 0))
  _ _ _ -> (error "negative fee, balance, or amount"))

(define t.configure
  {number --> (number --> t.action --> t.result)}
  Fee -> (/. Balance (/. Action (t.apply Fee Balance Action))) where (>= Fee 0)
  _ -> (error "negative fee"))

(define t.assert
  {boolean --> boolean}
  true -> true
  false -> (error "transition assertion failed"))

(let LowFee (t.configure 2)
  (let HighFee (t.configure 5)
    (let Before 100
      (do (t.assert (= (LowFee Before [debit 10]) [t-ok 88]))
          (t.assert (= (HighFee Before [debit 10]) [t-ok 85]))
          (t.assert (= (LowFee Before [credit 10]) [t-ok 110]))
          (t.assert (= (HighFee Before [credit 10]) [t-ok 110]))
          (t.assert (= (LowFee Before [debit 10]) (LowFee Before [debit 10])))
          (t.assert (= Before 100))
          (t.assert (= (LowFee 11 [debit 10]) insufficient-funds))
          (t.assert (= (HighFee 14 [debit 10]) insufficient-funds))
          (t.assert (= (LowFee 12 [debit 10]) [t-ok 0]))
          (t.assert (= (LowFee Before [debit 10]) [t-ok 88]))))))
```

Here is a deliberately different abstraction: a factory allocates a private
one-cell vector, returning only a function that increments it. Calls take a dummy
symbol, so creation and invocation are distinct. This block is typed too; vector
mutation is not hidden behind an asserted pure operation type.

```shen
(define t.counter
  {--> (symbol --> number)}
  -> (let Cell (vector 1)
       (do (vector-> Cell 1 0)
           (/. Tick
             (let Next (+ (<-vector Cell 1) 1)
               (do (vector-> Cell 1 Next) Next))))))

(let First (t.counter)
  (let Alias First
    (let Second (t.counter)
      (do (t.assert (= (First tick) 1))
          (t.assert (= (Alias tick) 2))
          (t.assert (= (Second tick) 1))
          (t.assert (= (First tick) 3))
          (t.assert (= (Second tick) 2))))))
```

Aliases of a single returned function share its cell; a new factory invocation
allocates another cell. Privacy is lexical encapsulation (no vector is returned),
not a capability theorem against runtime introspection. Tests assume sequential
execution; no concurrency or atomicity guarantee is made.

Both account transition and counter have arrow types. Such types say nothing
about mutation, replay, aliasing, fee immutability, arithmetic laws, or allocation
freshness. Those properties come from the respective bodies, closure/vector
semantics, and behavioral assertions. In particular the transition never updates
the predecessor balance, and no call can replace the captured fee. No trusted
declarations or untyped fences are needed; primitives and runtime semantics
remain trusted. Types do not prove overdraft policy or nonnegative results.
