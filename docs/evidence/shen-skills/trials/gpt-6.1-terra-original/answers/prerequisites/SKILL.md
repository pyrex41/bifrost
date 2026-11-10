# Cyclic prerequisites

The facts are the authoritative edges. The path relation remembers visited nodes, so a finite graph and a ground starting job bound recursion. Negation here is a ground membership test, not classical negation over unknowns. `defprolog` and its entry point are untyped; the declaration is trusted.

<!-- untyped -->
```shen
(defprolog edge
  compile seed <--;
  package compile <--;
  publish package <--;
  seed package <--;)
(defprolog path
  X Y Seen <-- (edge X Y) (when (not (element? Y Seen)));
  X Y Seen <-- (edge X Z) (when (not (element? Z Seen))) (path Z Y [Z | Seen]);)
(define raw-requirements X -> (prolog? (findall Y (path (receive X) Y [(receive X)]) Ys) (return Ys)))
(declare raw-requirements [symbol --> [list symbol]])
```

```shen
(define distinct-p {(list A) --> (list A)}
  [] -> []
  [X | Xs] -> (distinct-p Xs) where (element? X Xs)
  [X | Xs] -> [X | (distinct-p Xs)])
(define requirements {symbol --> (list symbol)}
  X -> (distinct-p (raw-requirements X)))
(define subset-p {(list symbol) --> (list symbol) --> boolean}
  [] _ -> true
  [X | Xs] B -> (and (element? X B) (subset-p Xs B)))
(define same-set {(list symbol) --> (list symbol) --> boolean}
  A B -> (and (= (length A) (length B)) (subset-p A B)))
(define check-p {boolean --> boolean} true -> true false -> (error "prerequisite assertion failed"))
(check-p (same-set (requirements publish) [package compile seed]))
(check-p (same-set (requirements seed) [package compile]))
(check-p (= (requirements docs) []))
```

Deduplication is explicit; simple paths suffice for reachability because cycles can be removed from any path. This does not establish correctness for nonground Prolog queries or infinite graphs.
