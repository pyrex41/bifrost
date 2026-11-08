# Prolog search with explicit query contracts

Read this for relational queries, negation as failure, cyclic graph data, or a typed caller of Prolog.

- [Facts, modes and cycle handling](#facts-modes-and-cycle-handling)
- [A small trusted boundary](#a-small-trusted-boundary)
- [Execution choices](#execution-choices)

## Facts, modes and cycle handling

Keep catalog membership independent of prerequisite edges: `art100` is an isolated course and is still
eligible. The `loop-a` / `loop-b` fixture exercises cycle handling. Whether such a cycle is legal catalog
data is a separate policy; the search must have defined behavior either way.

The supported calls supply a ground student or course. `course` grounds `C` before the negative goals;
`requires` grounds `P` before checking completion. Reordering these goals changes the computation.
Eligibility here means that every **direct** prerequisite is recorded as completed.

Put this first block in an untyped module. `defprolog` has no signature syntax equivalent to a typed
`define`; typed callers below use explicitly declared wrappers.

<!-- untyped -->
```shen
(defprolog course
  cs101 <--; cs201 <--; cs220 <--; cs301 <--; cs350 <--;
  math210 <--; art100 <--; loop-a <--; loop-b <--;)

(defprolog requires
  cs201 cs101 <--;
  cs220 cs101 <--;
  cs301 cs201 <--;
  cs301 cs220 <--;
  cs350 cs301 <--;
  cs350 math210 <--;
  loop-a loop-b <--;
  loop-b loop-a <--;)

(defprolog completed
  ada cs101 <--; ada cs201 <--; ada math210 <--;
  bob cs101 <--;)

(defprolog not-provable
  P <-- (call P) ! (when false);
  _ <--;)

(defprolog missing
  S C <-- (requires C P) (not-provable (completed S P));)

(defprolog can-take
  S C <-- (course C) (not-provable (completed S C)) (not-provable (missing S C));)

(defprolog prerequisite-path
  C P Seen <-- (requires C P) (when (not (element? P Seen)));
  C P Seen <-- (requires C Q) (when (not (element? Q Seen)))
               (prerequisite-path Q P [Q | Seen]);)

(defprolog needs
  C P <-- (prerequisite-path C P [C]);)

(define symbol-list?
  [] -> true
  [X | Xs] -> (and (symbol? X) (symbol-list? Xs))
  _ -> false)

(define checked-symbol-list
  X -> X where (symbol-list? X)
  _ -> (error "Prolog returned a non-symbol list"))

(define prerequisites
  C -> (checked-symbol-list
         (prolog? (findall P (needs (receive C) P) Ps) (return Ps))))

(define eligible
  S -> (checked-symbol-list
         (prolog? (findall C (can-take (receive S) C) Cs) (return Cs))))

\\ Trusted declarations: the checker has not verified these wrapper bodies.
(declare prerequisites [symbol --> [list symbol]])
(declare eligible [symbol --> [list symbol]])

(if (= (prolog? (not-provable (completed ada C))) false)
    passed
    (error "negation with an unbound variable was misinterpreted"))
```

`not-provable` tests whether a goal can be proved; it does not enumerate its complement. The last check
fails to prove that Ada completed *nothing*, because she has completed some courses. To enumerate courses
she has not completed, first enumerate ground courses and then negate each completion query.

`prerequisite-path` enumerates simple paths to other courses. Its visited list excludes the requested
course even when a cycle leads back to it. Each recursive step adds a previously unseen vertex, so a
finite edge relation gives finite search. Different paths can still find the same prerequisite. This
is an explicit domain policy, not the unrestricted mathematical transitive closure of a cyclic graph.

## A small trusted boundary

The wrappers validate their result shape at run time. Their `declare`s are still trusted assertions:
typechecking the caller does not retroactively check Prolog or prove the validator correct. Audit the
ground facts, query modes, validator and declarations together. A changed fact representation can
invalidate that contract.

The next block loads with `(tc +)` enabled. Check membership and count rather than depending on solution
order; `findall` preserves multiple derivations, and `nub` implements the chosen set view explicitly.

```shen
(define nub
  {(list A) --> (list A)}
  [] -> []
  [X | Xs] -> (nub Xs) where (element? X Xs)
  [X | Xs] -> [X | (nub Xs)])

(define prolog-check
  {string --> boolean --> symbol}
  _ true -> passed
  Label false -> (error "Prolog check failed: ~A" Label))

(define ada-eligible?
  {--> boolean}
  -> (let Courses (nub (eligible ada))
       (and (= (length Courses) 2)
            (element? cs220 Courses)
            (element? art100 Courses))))

(define cs350-prerequisites?
  {--> boolean}
  -> (let Courses (nub (prerequisites cs350))
       (and (= (length Courses) 5)
            (element? cs301 Courses)
            (element? math210 Courses)
            (element? cs201 Courses)
            (element? cs220 Courses)
            (element? cs101 Courses))))

(prolog-check "isolated catalog course is eligible" (ada-eligible?))
(prolog-check "transitive prerequisites" (cs350-prerequisites?))
(prolog-check "cycle terminates and excludes origin"
  (= (nub (prerequisites loop-a)) [loop-b]))
(prolog-check "isolated course has no prerequisites" (= (prerequisites art100) []))
```

A typed caller must still satisfy the declared interface:

<!-- reject: wrong-student -->
```shen
(define wrong-student
  {--> (list symbol)}
  -> (eligible 42))
```

## Execution choices

Before choosing or changing a relation, record the following:

| Question | Consequence |
|---|---|
| Which arguments are ground on entry? | Negation and functional tests need suitable values; a mode declaration changes matching/unification behavior but does not prove termination. |
| Is the search finite in that mode? | `findall` requires exhaustion of the search. A two-clause recursive reachability relation can loop on cycles. |
| What are clause and goal orders? | They affect depth-first search, work performed, and whether later answers are reached. |
| Are cuts justified? | `!` removes alternatives and can change which answers exist operationally. Test those answers explicitly. |
| Are answers a sequence, multiset or set? | Choose ordering and duplicate policies at the public boundary. |
| Which absence assumption is intended? | Negation as failure assumes that failure to prove the ground goal is the relevant notion of absence. |

Use `receive` for Shen values entering a query and `return` for a result leaving it. `return` terminates
that Prolog computation; `findall` collects all answers. `when` runs a Shen boolean expression, while
`call` and `fork` support higher-order goals and alternative goals. Consult the official modes and
occurs-check documentation before changing unification for performance.

For large graphs, a worklist, visited set, memoisation or index may fit the required query better than
enumerating all simple paths. Associations and other ordinary data structures are valid representations;
choose the representation and algorithm by semantics and cost.

For changing relational facts, Shen also supplies `asserta`, `assertz` and `retract`. Follow their dynamic
predicate regime: the book warns that defining a predicate through `defprolog` disables that dynamic
update route. A custom data interpreter is another choice, especially when the rules themselves need
validation, persistence or multiple interpretations.

## Sources

- TBoS, online fifth edition, [25.6, Shen Prolog](https://shenlanguage.org/TBoS/364.html), especially
  [return/findall](https://shenlanguage.org/TBoS/365.html),
  [cuts and negation as failure](https://shenlanguage.org/TBoS/368.html), and
  [modes and dynamic predicates](https://shenlanguage.org/TBoS/369.html).
- [ShenDoc: Prolog](https://shenlanguage.org/SD/Prolog.html) for the query interface, modes and unification.
- The [accompanying Prolog programs](https://shenlanguage.org/TBoS/Programs/25.6.txt). Program download
  numbering elsewhere on the site may differ from the current edition's prose; cite both title and URL.
