# Immutable policy, explicit account state

A configuration captures only a nonnegative debit fee. Each call supplies the current balance and an action. A successful result contains the new balance; insufficient funds is an explicit alternative, not a mutated balance or an exception. Negative configuration, balance, or amount is invalid and raises an error. Tests use exact small integers; arrow types admit numbers more broadly.

```shen
(datatype a-action
  N : number;
  =====================
  [credit N] : a-action;

  N : number;
  ====================
  [debit N] : a-action;)

(datatype a-result
  N : number;
  ==========================
  [balance N] : a-result;

  ______________________________
  insufficient-funds : a-result;)

(define a-credit
  {number --> number --> a-result}
  Balance Amount -> [balance (+ Balance Amount)]
    where (and (>= Balance 0) (>= Amount 0))
  _ _ -> (simple-error "negative balance or credit"))

(define a-debit
  {number --> number --> number --> a-result}
  Fee Balance Amount ->
    (if (>= Balance (+ Amount Fee))
        [balance (- Balance (+ Amount Fee))]
        insufficient-funds)
    where (and (>= Balance 0) (>= Amount 0))
  _ _ _ -> (simple-error "negative balance or debit"))

(define a-step
  {number --> number --> a-action --> a-result}
  _ Balance [credit Amount] -> (a-credit Balance Amount)
  Fee Balance [debit Amount] -> (a-debit Fee Balance Amount))

(define a-configure
  {number --> (number --> a-action --> a-result)}
  Fee -> (a-step Fee) where (>= Fee 0)
  _ -> (simple-error "negative debit fee"))

(define a-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(a-check "independent fee configurations and replay"
  (let Low (a-configure 1)
       High (a-configure 4)
    (= [(Low 20 [debit 5]) (High 20 [debit 5]) (Low 20 [debit 5])]
       [[balance 14] [balance 11] [balance 14]])))
(a-check "credit does not pay debit fee"
  (= ((a-configure 4) 20 [credit 5]) [balance 25]))
(a-check "predecessor remains unchanged and reusable"
  (let Before 20
       Step (a-configure 1)
       After (Step Before [debit 5])
    (and (= Before 20)
      (and (= After [balance 14]) (= (Step Before [credit 2]) [balance 22])))))
(a-check "overdraft includes fee"
  (= ((a-configure 2) 10 [debit 9]) insufficient-funds))
(a-check "exact funds allowed"
  (= ((a-configure 2) 10 [debit 8]) [balance 0]))
(a-check "failed transition has no hidden state"
  (let Step (a-configure 2)
       Failure (Step 10 [debit 99])
    (and (= Failure insufficient-funds) (= (Step 10 [debit 8]) [balance 0]))))
```

## Separately: genuinely private mutation

The counter's cell is allocated once per factory call and is never returned. Aliases of a returned closure share it. These counters are independent of account transitions.

```shen
(define a-counter-command
  {(vector number) --> symbol --> number}
  Cell peek -> (<-vector Cell 1)
  Cell tick -> (let Next (+ 1 (<-vector Cell 1))
                   Write (vector-> Cell 1 Next)
                Next)
  _ _ -> (simple-error "unknown counter command"))

(define a-counter
  {number --> (symbol --> number)}
  Start -> (let Cell (vector 1)
               Init (vector-> Cell 1 Start)
             (a-counter-command Cell)))

(a-check "aliases share, factories do not"
  (let C (a-counter 0)
       Alias C
       Other (a-counter 0)
       One (C tick)
       Shared (Alias peek)
       Two (Alias tick)
    (= [One Shared Two (C peek) (Other peek) (Other tick) (C peek)]
       [1 1 2 2 0 1 2])))
```

All bodies are checked, without declarations or untyped boundaries. The action/result rules specify concrete shapes. They do not encode nonnegative amounts, conservation, replay, isolation, purity, ownership, or any invocation protocol. Configuration isolation and replay follow by inspecting the account body: only captured numeric Fee, explicit inputs and arithmetic are used. Counter sharing/freshness instead depends on allocation and vector mutation semantics, supported by assertions. Runtime arithmetic, immutable numeric values, evaluation order and stable built-ins remain assumptions. The fee is fixed per returned closure, not a claim that arbitrary captured objects are immutable. No serialization, concurrency or general effect proof is supplied.
