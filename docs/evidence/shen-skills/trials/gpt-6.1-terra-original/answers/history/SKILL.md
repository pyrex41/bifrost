# Indexed persistent history

The representation is a newest-first immutable list. The successor index records one more element than its predecessor. These are checked implementation bodies, not operation axioms.

```shen
(datatype hist
  ____ [] : (hist z A);
  X : A; Xs : (hist N A);
  =========================
  [X | Xs] : (hist (s N) A);)

(define empty-h {--> (hist z A)} -> [])
(define add-h {A --> (hist N A) --> (hist (s N) A)} X H -> [X | H])
(define newest {(hist (s N) A) --> A} [X | _] -> X)
(define remove-h {(hist (s N) A) --> (hist N A)} [_ | H] -> H)
(define double-h {(hist (s N) number) --> (hist (s N) number)}
  [X | H] -> [(* 2 X) | H])
(define check-h {boolean --> boolean} true -> true false -> (error "history assertion failed"))
(check-h (= (newest (add-h 8 (add-h 3 (empty-h)))) 8))
(check-h (= (newest (add-h 3 (empty-h))) 3))
(check-h (= (remove-h (double-h (add-h 8 (add-h 3 (empty-h))))) (add-h 3 (empty-h))))
(check-h (= (newest (double-h (add-h 8 (add-h 3 (empty-h))))) 16))
```

<!-- reject: empty-read -->
```shen
(define empty-read {--> number} -> (newest (empty-h)))
```

<!-- reject: removed-read -->
```shen
(define removed-read {--> number} -> (newest (remove-h (add-h 1 (empty-h)))))
```

<!-- reject: false-growth -->
```shen
(define false-growth {A --> (hist N A) --> (hist (s N) A)} X H -> H)
```

The tests illustrate persistence and predecessor preservation; the type index alone does not prove value ordering or purity in arbitrary code.
