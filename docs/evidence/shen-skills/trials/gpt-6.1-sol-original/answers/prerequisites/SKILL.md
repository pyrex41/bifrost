# Cyclic prerequisites

Use Prolog facts and simple-path search, not unrestricted transitive recursion.
`Seen` initially contains the queried job. Each edge extends it, so neither the
input itself nor any already visited vertex can be returned on that path.

`defprolog` is an explicitly untyped boundary in Shen. The single exported
`q.raw` declaration below is trusted; its symbol/list contract is not inferred
from the Prolog body. Call it only with a ground symbol. The finite static facts
and the groundness discipline are part of the termination argument.

<!-- untyped -->
```shen
(defprolog q.edge
  compile seed <--;
  package compile <--;
  publish package <--;
  seed package <--;)

(defprolog q.path
  X Y Seen <-- (q.edge X Y)
               (when (not (element? Y Seen)));
  X Y Seen <-- (q.edge X Z)
               (when (not (element? Z Seen)))
               (q.path Z Y [Z | Seen]);)

(define q.raw
  Job -> (prolog? (findall R (q.path (receive Job) R [(receive Job)]) Rs)
                 (return Rs)))

(declare q.raw [symbol --> [list symbol]])
```

```shen
(define q.nub
  {(list A) --> (list A)}
  [] -> []
  [X | Xs] -> (q.nub Xs) where (element? X Xs)
  [X | Xs] -> [X | (q.nub Xs)])

(define q.requirements
  {symbol --> (list symbol)}
  Job -> (q.nub (q.raw Job)))

(define q.subset?
  {(list A) --> (list A) --> boolean}
  [] _ -> true
  [X | Xs] Ys -> (and (element? X Ys) (q.subset? Xs Ys)))

(define q.assert
  {boolean --> boolean}
  true -> true
  false -> (error "prerequisite assertion failed"))

(define q.expect
  {symbol --> (list symbol) --> boolean}
  Job Expected ->
    (let Got (q.requirements Job)
      (do (q.assert (q.subset? Got Expected))
          (q.assert (q.subset? Expected Got))
          (q.assert (= (length Got) (length Expected)))
          (q.assert (not (element? Job Got))))))

(q.expect publish [package compile seed])
(q.expect seed [package compile])
(q.expect docs [])
```

`docs` is an isolated job: it needs no edge fact and has no reachable vertex.
Other unknown ground symbols likewise yield `[]`; this interface does not
validate membership in a closed job-name enumeration.
Output order is intentionally unspecified. `nub` removes duplicates across
paths, while `Seen` prevents repeats within a path. A finite graph has only
finitely many simple paths; every reachable distinct vertex has a simple path,
so restricting traversal this way remains complete, including in this cycle.
The tests compare sets and cardinalities, not engine enumeration order.

The only negation is the Shen Boolean `not` on `element?`, after the edge has
bound its endpoint and with a ground `Seen` list. It is a decidable ground
membership test, not a claim about unbound Prolog variables or an open-world
absence of edges. Unground/reverse queries are not supported by this interface.
Types do not prove graph reachability, finiteness, or declaration soundness.
