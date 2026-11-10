# Cyclic prerequisite closure

Use a typed explicit worklist rather than unrestricted recursive Prolog closure: finite graph traversal needs a visited set regardless of engine. The independent catalog includes isolated docs. Input must be a ground catalog symbol; an unknown symbol raises an error. Ordinary evaluated Shen calls supply values, not Prolog logic variables.

```shen
(define p-jobs
  {--> (list symbol)}
  -> [seed compile package publish docs])

(define p-direct
  {symbol --> (list symbol)}
  seed -> [package]
  compile -> [seed]
  package -> [compile]
  publish -> [package]
  docs -> []
  _ -> (simple-error "unknown job"))

(define p-walk
  {(list symbol) --> (list symbol) --> (list symbol) --> (list symbol)}
  [] _ Answer -> (reverse Answer)
  [J | Todo] Seen Answer -> (p-walk Todo Seen Answer)
    where (element? J Seen)
  [J | Todo] Seen Answer ->
    (p-walk (append (p-direct J) Todo) [J | Seen] [J | Answer]))

(define p-requirements
  {symbol --> (list symbol)}
  J -> (p-walk (p-direct J) [J] []) where (element? J (p-jobs))
  _ -> (simple-error "input must be a catalog job"))

(define p-unique?
  {(list symbol) --> boolean}
  [] -> true
  [X | Xs] -> (and (not (element? X Xs)) (p-unique? Xs)))

(define p-all?
  {(A --> boolean) --> (list A) --> boolean}
  _ [] -> true
  F [X | Xs] -> (and (F X) (p-all? F Xs)))

(define p-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(p-check "publish closure" (= (p-requirements publish) [package compile seed]))
(p-check "cycle excludes starting seed" (= (p-requirements seed) [package compile]))
(p-check "isolated catalog job" (= (p-requirements docs) []))
(p-check "all catalog closures are unique and exclude their input"
  (p-all? (/. J (and (p-unique? (p-requirements J))
                    (not (element? J (p-requirements J)))))
          (p-jobs)))
```

`Seen` initially contains the input, so even a cycle cannot return it. Each newly visited vertex schedules all its outgoing prerequisites once. Every reachable non-input vertex is eventually processed: skipping a seen vertex loses no successors because they were already scheduled, except for the root whose successors were scheduled initially. Only five vertices can expand; each expansion adds a finite list and each iteration removes a work item. Thus traversal terminates and the accumulated answer contains each reachable non-input job once, in first-visit order.

The tests assert exact ordered lists, not just counts. `not` is Boolean negation of a ground membership test, not negation-as-failure on an unbound query. This is a closed finite graph; changes must keep all edge endpoints in the catalog. Signatures establish list/symbol shapes, not reachability correctness or catalog consistency. No declarations or untyped boundaries.
