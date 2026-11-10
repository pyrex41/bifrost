# Indices, abstraction, and composition

Sections: [indexed sequences](#indexed-sequences), [abstract operations](#abstract-operations), and
[composition and its assumptions](#composition-and-its-assumptions). Run this file as its own sequence;
only the explicitly marked implementation block is loaded without type checking.

## Indexed sequences

Use an index to carry a fact the caller needs. Here `z` and `(s N)` are symbolic size terms; they are
not arithmetic expressions evaluated by the checker. A successor index rules out the empty case for
these head and tail operations.

```shen
(datatype seq
  ________________
  [] : (seq z A);

  X : A; Xs : (seq N A);
  ==========================
  [X | Xs] : (seq (s N) A);

  Xs : (list A) >> P;
  ____________________
  Xs : (seq N A) >> P;)

(define seq-head
  {(seq (s N) A) --> A}
  [X | _] -> X)

(define seq-tail
  {(seq (s N) A) --> (seq N A)}
  [_ | Xs] -> Xs)

(define seq-size
  {(seq N A) --> number}
  Xs -> (length Xs))

(if (= (seq-head (seq-tail [a b c])) b)
    true
    (simple-error "indexed access changed"))

(if (= (seq-size [a b c]) 3)
    true
    (simple-error "sequence forgetting changed"))
```

The invalid client asks for a successor index after removing the only element:

<!-- reject: empty-seq-head -->
```shen
(define empty-seq-head
  {--> symbol}
  -> (seq-head (seq-tail [a])))
```

The last datatype rule forgets the index for a proof using ordinary list operations. This is useful
when the client does not need to preserve length in its result type.

Do not read this example as GADT-style refinement of an arbitrary signature variable by case
analysis. This encoding does not provide that behavior. More elaborate indexed algorithms may need
additional rules and inductive facts about their indices; the automatic checker does not invent
arbitrary induction proofs. A sequent judgment can express dependencies that ordinary arrow
signatures cannot express. Distinguish that expressive power from automatic provability.

Source: TBoS [§20.8, Dependent Types](https://shenlanguage.org/TBoS/270.html), including the
indexed append and type-valued input discussion on [p.257](https://shenlanguage.org/TBoS/271.html).
For indexed interfaces over persistent closures, read
[persistent stacks and indexed interfaces](../../shen-thinking/references/indexed-stacks.md).

## Abstract operations

Hide a representation when callers should depend on its operations and laws. A small untyped
implementation can be exposed through reviewed operation axioms, as in TBoS
[§§21.1–21.2](https://shenlanguage.org/TBoS/274.html). These axioms are trusted; they do not verify
the implementation. Keep that boundary visible.

The following implementation uses lists, but publishes only the abstract interface below:

<!-- untyped -->
```shen
(define bag-of X -> [X])
(define bag-add X B -> [X | B])
(define bag-size B -> (length B))
```

```shen
(datatype bag
  ___________________________
  bag-of : (A --> (bag A));

  _______________________________________
  bag-add : (A --> (bag A) --> (bag A));

  ________________________________
  bag-size : ((bag A) --> number);)

(define three-bag
  {--> (bag symbol)}
  -> (bag-add a (bag-add b (bag-of c))))

(define retain-bag
  {(bag A) --> (list (bag A))}
  B -> [B])

(if (= (bag-size (head (retain-bag (three-bag)))) 3)
    true
    (simple-error "abstract bag storage changed"))
```

The next observation separates representation equality from a possible bag interpretation: the
same two elements inserted in opposite order have equal size but unequal list representations. A real
semantic bag equality must also account for multiplicity; equal size alone is not enough.

```shen
(define bag-order-observation
  {--> boolean}
  -> (let AB (bag-add a (bag-of b))
          BA (bag-add b (bag-of a))
       (and (= (bag-size AB) (bag-size BA)) (not (= AB BA)))))

(if (bag-order-observation)
    true
    (simple-error "bag representation equality observation changed"))
```

An abstract value can be stored and passed through polymorphic code. It is the representation-specific
operation that is unavailable without a suitable rule:

<!-- reject: peek-bag -->
```shen
(define peek-bag
  {(bag symbol) --> symbol}
  B -> (head B))
```

This interface boundary governs typed clients; untyped code can still inspect the representation.
If the same public names also receive list signatures, those signatures remain another usable interface.

Generic printing is possible but may expose representation details. Generic `=` permits operands of
the same type, yet it need not implement the abstraction's intended equality, especially for a
closure representation. Publish operations such as `bag->string` and `bag=?` when those observations
must have stable meaning. See TBoS [§17.7, Equality Types](https://shenlanguage.org/TBoS/216.html)
and the [official equality reference](https://shenlanguage.org/SD/Equality.html).

## Composition and its assumptions

Give a combinator the contract required of its argument. The checker can then verify that the
combinator feeds each result to an operation that accepts it:

```shen
(define twice-bag
  {((bag A) --> (bag A)) --> (bag A) --> (bag A)}
  F B -> (F (F B)))

(if (= (bag-size (twice-bag (bag-add a) (bag-of b))) 3)
    true
    (simple-error "abstract update composition changed"))
```

The claim established here is preservation of the abstract type, relative to the operation axioms.
A list function with no corresponding abstract contract cannot be substituted just because the
current representation happens to be a list:

<!-- reject: raw-bag-update -->
```shen
(define raw-bag-update
  {(bag symbol) --> (bag symbol)}
  B -> (twice-bag (fn reverse) B))
```

State semantic laws separately. For this implementation, `bag-size (bag-of X) = 1` and
`bag-size (bag-add X B) = 1 + bag-size B` describe useful behavior; the abstract arrow signatures do
not prove those equations. Review or prove the implementations, and test observable laws and their
composition. A replacement operation could preserve `(bag A)` while violating a size or membership
law.

TBoS [§21.4, Algebraic Simplification](https://shenlanguage.org/TBoS/282.html) applies the same
architectural pattern to admitted transformations and typed traversal/composition. Keep the trusted
primitive laws separate from what the checker derives about the combinators. A semantic-preservation
claim requires those laws; a label on the result type alone does not establish equality to a particular
input expression. General termination, mutable-state effects, and lifetime invariants need additional
arguments as well.
