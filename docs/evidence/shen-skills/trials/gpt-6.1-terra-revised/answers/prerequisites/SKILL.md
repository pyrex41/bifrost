# Cyclic prerequisites

A finite functional graph and a visited worklist avoid Prolog's depth-first cycle hazard. The input is a ground job symbol; unknown symbols have no outgoing edges. `element?` is ordinary ground membership, not negation-as-failure. Each first visit expands its outgoing edges once; origin starts visited, so it is excluded even on a cycle. Returned order is traversal order, not a specification.

```shen
(define edges {symbol --> (list symbol)}
 seed -> [package]
 compile -> [seed]
 package -> [compile]
 publish -> [package]
 _ -> [])
(define walk {(list symbol) --> (list symbol) --> (list symbol)}
 [] Seen -> Seen
 [X | Xs] Seen -> (walk Xs Seen) where (element? X Seen)
 [X | Xs] Seen -> (walk (append (edges X) Xs) [X | Seen]))
(define requirements {symbol --> (list symbol)}
 J -> (remove J (walk (edges J) [J])))
(define qcheck {string --> boolean --> boolean}
 _ true -> true
 L false -> (simple-error L))
(qcheck "publish" (= (requirements publish) [seed compile package]))
(qcheck "seed cycle" (= (requirements seed) [compile package]))
(qcheck "docs" (= (requirements docs) []))
```

Completeness follows by induction on paths: every newly reached vertex is expanded, while already expanded vertices need not be expanded again. Finite reachable vertices and finite edge lists ensure termination here. These facts depend on the closed edge definition and ground finite inputs; the arrow type itself does not establish them.
