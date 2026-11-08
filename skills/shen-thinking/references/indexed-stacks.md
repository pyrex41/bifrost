# Persistent stacks and indexed interfaces

Use this reference when implementing an abstract value with closures or specifying which operations
are legal at a particular structural index. It develops the same stack contract with a closure
representation and a list representation. The public names differ so both can be tested together.

## Contents

- [Start with the laws](#start-with-the-laws)
- [Closure representation](#closure-representation)
- [Indexed operation judgments](#indexed-operation-judgments)
- [Rejected clients](#rejected-clients)
- [A checked list representation](#a-checked-list-representation)
- [Compare observations](#compare-observations)
- [What has been established](#what-has-been-established)
- [Sources](#sources)

## Start with the laws

The central equations, for a valid stack `s` and element `x`, are:

```text
top(push(x, s)) = x
pop(push(x, s)) = s
```

`top` and `pop` require a nonempty stack. Pushing creates a new value; popping returns a predecessor
without modifying either value. This persistence keeps existing aliases and their length facts valid.
The discussion concerns the stack structure: an element may itself refer to mutable storage unless
another invariant excludes that.

## Closure representation

Load this representation deliberately with checking off. The next block supplies its asserted logical
interface. The `untyped` annotation is understood by `bifrost skills check`; when loading manually,
switch checking off before this file and restore it before the interface/client file.

<!-- untyped -->
```shen
(define cs-empty
  -> empty-closure-stack)

(define cs-dispatch
  X _ top -> X
  _ S pop -> S
  _ _ _ -> (simple-error "unknown closure-stack operation"))

(define cs-push
  X S -> (cs-dispatch X S))

(define cs-top
  S -> (S top))

(define cs-pop
  S -> (S pop))
```

`cs-dispatch` takes three arguments. Supplying `X` and `S` produces a function that retains the element
and predecessor, waiting for the requested operation. No explicit lambda is needed. The empty marker
is intentionally not callable; typed clients will have no rule allowing `top` or `pop` at length zero.
Untyped callers can bypass that boundary and cause runtime errors.

Follow the equations by unfolding the definitions:

```text
cs-top(cs-push(x, s)) = cs-dispatch(x, s, top) = x
cs-pop(cs-push(x, s)) = cs-dispatch(x, s, pop) = s
```

These are informal reduction arguments for these particular bodies. They are separate from the client
typechecking that follows.

## Indexed operation judgments

`(closure-stack A N)` means a stack of elements of type `A` with length index `N`. The empty index is
`0`; `[succ N]` is a structural successor term, not arithmetic on a runtime length field.

```shen
(datatype closure-stack
  ______________________________
  (cs-empty) : (closure-stack A 0);

  X : A; S : (closure-stack A N);
  __________________________________________
  (cs-push X S) : (closure-stack A [succ N]);

  S : (closure-stack A [succ N]);
  ______________________________
  (cs-top S) : A;

  S : (closure-stack A [succ N]);
  ______________________________
  (cs-pop S) : (closure-stack A N);)

(define stack-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(define cs-observe-two
  {(closure-stack A [succ [succ N]]) --> (A * A)}
  S -> (@p (cs-top S) (cs-top (cs-pop S))))

(stack-check "closure top and predecessor laws"
  (= (cs-observe-two (cs-push 20 (cs-push 10 (cs-empty))))
     (@p 20 10)))

(stack-check "pop preserves the original alias"
  (= (let S (cs-push 20 (cs-push 10 (cs-empty)))
          P (cs-pop S)
       [ (cs-top S) (cs-top P) (cs-top S) ])
     [20 10 20]))
```

Read each rule as a condition on an operation: push adds a successor index; top needs one; pop removes
one. The shared `A` connects all elements. The checker can combine these judgments without exposing
the closure representation to a client.

These are **trusted operation axioms** over untyped definitions. They do not ask the checker to inspect
those definitions or prove their equations. Keep the axioms and implementation together for review.
The [types skill](../../shen-types/SKILL.md) explains concrete rules, modes and proof search in more detail.

## Rejected clients

Each block isolates one invalid operation in a named client. The annotation also requires its name in
the type-error diagnostic, so an unrelated missing definition cannot satisfy the expected rejection.

<!-- reject: cs-top-empty -->
```shen
(define cs-top-empty
  {--> number}
  -> (cs-top (cs-empty)))
```

<!-- reject: cs-pop-empty -->
```shen
(define cs-pop-empty
  {--> (closure-stack number 0)}
  -> (cs-pop (cs-empty)))
```

<!-- reject: cs-top-after-pop -->
```shen
(define cs-top-after-pop
  {--> number}
  -> (cs-top (cs-pop (cs-push 10 (cs-empty)))))
```

<!-- reject: cs-mixed-elements -->
```shen
(define cs-mixed-elements
  {--> (closure-stack number [succ [succ 0]])}
  -> (cs-push tea (cs-push 10 (cs-empty))))
```

Rejection is relative to the active datatype environment. Shen's rules are extensible; adding an axiom
that admits an empty top would change the interface's guarantees.

## A checked list representation

The equivalent length contract can be attached to an ordinary list. Here a double-line rule provides
both construction and decomposition, and the representation functions themselves are checked:

```shen
(datatype list-stack
  _______________________
  [] : (list-stack A 0);

  X : A; S : (list-stack A N);
  ====================================
  [X | S] : (list-stack A [succ N]);)

(define ls-empty
  {--> (list-stack A 0)}
  -> [])

(define ls-push
  {A --> (list-stack A N) --> (list-stack A [succ N])}
  X S -> [X | S])

(define ls-top
  {(list-stack A [succ N]) --> A}
  [X | _] -> X)

(define ls-pop
  {(list-stack A [succ N]) --> (list-stack A N)}
  [_ | S] -> S)

(define ls-observe-two
  {(list-stack A [succ [succ N]]) --> (A * A)}
  S -> (@p (ls-top S) (ls-top (ls-pop S))))

(stack-check "list top and predecessor laws"
  (= (ls-observe-two (ls-push 20 (ls-push 10 (ls-empty))))
     (@p 20 10)))

(stack-check "list pop preserves the original alias"
  (= (let S (ls-push 20 (ls-push 10 (ls-empty)))
          P (ls-pop S)
       [(ls-top S) (ls-top P) (ls-top S)])
     [20 10 20]))
```

The checker rejects a body that claims to increase the length while returning its input unchanged:

<!-- reject: ls-broken-push -->
```shen
(define ls-broken-push
  {A --> (list-stack A N) --> (list-stack A [succ N])}
  _ S -> S)
```

It also rejects invalid clients of the checked representation:

<!-- reject: ls-top-empty -->
```shen
(define ls-top-empty
  {--> number}
  -> (ls-top (ls-empty)))
```

<!-- reject: ls-mixed-elements -->
```shen
(define ls-mixed-elements
  {--> (list-stack number [succ [succ 0]])}
  -> (ls-push tea (ls-push 10 (ls-empty))))
```

## Compare observations

Test the same observable operations on both representations. Do not use closure identity as a test of
abstract stack equality:

```shen
(define stack-observations-agree?
  {number --> number --> boolean}
  X Y -> (= (cs-observe-two (cs-push X (cs-push Y (cs-empty))))
            (ls-observe-two (ls-push X (ls-push Y (ls-empty))))))

(stack-check "representations agree on sample observations"
  (and (stack-observations-agree? 20 10)
       (and (stack-observations-agree? 0 0)
            (stack-observations-agree? -3 7))))
```

This is a small differential test. A representation-independence argument would define a relation
between corresponding closure and list values, show that empty values are related, and show that push,
top and pop preserve that relation or produce equal observations. Tests on three pairs do not establish
the universal claim.

## What has been established

| Layer | Closure representation | List representation |
|---|---|---|
| Representation code | Loaded untyped | Checked against concrete structural rules |
| Indexed interface | Asserted operation judgments | Checked signatures using the list datatype |
| Invalid clients | Empty access and mismatched element types rejected | Empty access and mismatched element types rejected |
| Behavioral laws | Informal reductions plus executable observations | Informal list reductions plus executable observations |
| Termination and effects | Reason about the bodies; not implied by the arrow types | Same limitation |

Changing the untyped `cs-push` body to return an unrelated symbol would leave its asserted operation
rule intact. Client typing could still succeed while execution fails. This is why an expressive
interface, an implementation that satisfies it, and faithful execution are separate obligations.

A destructive stack cannot inherit these aliasing claims just by reusing the same judgments. After a
mutating pop, another alias might still carry the old nonempty index. State an ownership, state-passing
or other justified discipline before claiming that interface is safe for mutation.

An operation that conditionally pushes or keeps its input also cannot promise a successor index on
every return. Represent its possible results with explicit alternatives or an appropriate indexed
judgment; see [tagged alternatives](../../shen-types/references/proof-search.md#tagged-alternatives).

## Sources

- Mark Tarver, [*The Book of Shen*, chapter 21](https://shenlanguage.org/TBoS/index.html), abstract
  operations and algebraic laws; [companion stack program](https://shenlanguage.org/TBoS/Programs/21.1.txt).
  The examples here add length indices, rejection checks and a paired representation.
- TBoS chapter 20 on proof control and dependent types, and the official
  [dependent types example](https://shenlanguage.org/OSM/Dependent.html).
- [Shen equality](https://shenlanguage.org/SD/Equality.html): distinguish supported generic comparison
  from an application's semantic equality contract.
