# Structurally indexed persistent history

Newest is the head of an immutable list; appending chronologically therefore constructs a new head. Index `0` is empty and `[succ N]` is nonempty. These are concrete representation rules, not axioms asserting operation results. Double-line rules allow decomposition, so all operation bodies below are checked.

```shen
(datatype h-history
  _________________________
  [] : (h-history A 0);

  X : A; P : (h-history A N);
  =======================================
  [X | P] : (h-history A [succ N]);)

(define h-empty
  {--> (h-history A 0)}
  -> [])

(define h-append
  {A --> (h-history A N) --> (h-history A [succ N])}
  X P -> [X | P])

(define h-newest
  {(h-history A [succ N]) --> A}
  [X | _] -> X)

(define h-remove
  {(h-history A [succ N]) --> (h-history A N)}
  [_ | P] -> P)

(define h-double-newest
  {(h-history number [succ N]) --> (h-history number [succ N])}
  H -> (h-append (* 2 (h-newest H)) (h-remove H)))

(define h-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(h-check "replacement, unchanged source and predecessor"
  (let P (h-append 7 (h-empty))
       H (h-append 11 P)
       Alias H
       R (h-double-newest H)
    (and (= (h-newest R) 22)
      (and (= (h-remove R) P)
        (and (= (h-newest (h-remove R)) 7)
          (and (= (h-newest Alias) 11)
               (= (h-remove H) P)))))))
(h-check "single-item replacement retains empty predecessor"
  (= (h-remove (h-double-newest (h-append 3 (h-empty)))) (h-empty)))
(h-check "removal leaves old history usable"
  (let H (h-append 4 (h-empty))
       P (h-remove H)
    (and (= P (h-empty)) (= (h-newest H) 4))))
```

<!-- reject: h-read-empty -->
```shen
(define h-read-empty
  {--> number}
  -> (h-newest (h-empty)))
```

<!-- reject: h-read-removed-sole -->
```shen
(define h-read-removed-sole
  {--> number}
  -> (h-newest (h-remove (h-append 5 (h-empty)))))
```

<!-- reject: h-false-growth -->
```shen
(define h-false-growth
  {A --> (h-history A N) --> (h-history A [succ N])}
  _ H -> H)
```

The replacement consumes a successor-indexed input, removes one head, and reconstructs exactly one head. Its signature checks length preservation; the body specifically reuses the predecessor, which is also tested. List reductions give newest(append(x,p)) = x and remove(append(x,p)) = p. No mutation is performed. These are implementation arguments, not consequences of arrow types alone. No untyped definitions or `declare`; trust remains in the concrete datatype theory, built-ins, and runtime. The datatype is open: adding inconsistent rules could invalidate rejection guarantees. Indices alone do not prove ordering, purity, termination, or arbitrary element immutability.
