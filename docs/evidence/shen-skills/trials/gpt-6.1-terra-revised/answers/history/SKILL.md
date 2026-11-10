# Persistent indexed history

A successor index describes each cons cell. The double-line constructor rule supports both construction and pattern decomposition; operation bodies are checked against it, rather than postulated. Indices establish shape under these open datatype rules, not purity or universal behavioral laws.

```shen
(datatype history
  ___________________
  [] : (history A 0);
  X : A; S : (history A N);
  =================================
  [X | S] : (history A [succ N]);)

(define h-empty {--> (history A 0)} -> [])
(define h-append {A --> (history A N) --> (history A [succ N])} X S -> [X | S])
(define h-newest {(history A [succ N]) --> A} [X | _] -> X)
(define h-remove {(history A [succ N]) --> (history A N)} [_ | S] -> S)
(define h-double {(history number [succ N]) --> (history number [succ N])}
  [X | S] -> [(* X 2) | S])
(define h-test {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))
(h-test "persistence and predecessor" 
  (= (let S (h-append 7 (h-append 3 (h-empty)))
          D (h-double S)
       [(h-newest S) (h-newest D) (h-newest (h-remove D)) (h-newest (h-remove S))])
     [7 14 3 3]))
```

<!-- reject: h-bad-growth -->
```shen
(define h-bad-growth {A --> (history A N) --> (history A [succ N])}
  _ S -> S)
```

<!-- reject: h-read-empty -->
```shen
(define h-read-empty {--> number} -> (h-newest (h-empty)))
```

<!-- reject: h-read-removed -->
```shen
(define h-read-removed {--> number} -> (h-newest (h-remove (h-append 1 (h-empty)))))
```
