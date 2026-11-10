# Indexed persistent history

Newest items live at the front of a plain immutable list. The index is a
structural term, not an arithmetic claim. These are representation rules and
checked implementation bodies, not axioms asserting operation signatures.

```shen
(datatype h.history
  __________________________
  [] : (h.history hz A);

  X : A; Xs : (h.history N A);
  ========================================
  [X | Xs] : (h.history (hs N) A);)

(define h.empty
  {--> (h.history hz A)}
  -> [])

(define h.append
  {A --> (h.history N A) --> (h.history (hs N) A)}
  X H -> [X | H])

(define h.newest
  {(h.history (hs N) A) --> A}
  [X | _] -> X)

(define h.remove
  {(h.history (hs N) A) --> (h.history N A)}
  [_ | H] -> H)

(define h.double-newest
  {(h.history (hs N) number) --> (h.history (hs N) number)}
  [X | H] -> [(* 2 X) | H])

(define h.assert
  {boolean --> boolean}
  true -> true
  false -> (error "history assertion failed"))

(let Old (h.append 7 (h.empty))
  (let New (h.append 11 Old)
    (let Replaced (h.double-newest New)
      (do (h.assert (= (h.newest Old) 7))
          (h.assert (= (h.newest New) 11))
          (h.assert (= (h.newest Replaced) 22))
          (h.assert (= (h.remove New) Old))
          (h.assert (= (h.remove Replaced) Old))
          (h.assert (= (h.remove (h.remove Replaced)) (h.empty)))
          (h.assert (= (h.newest (h.remove Replaced)) 7))))))
```

Reading empty is rejected:

<!-- reject: h.bad-empty-read -->
```shen
(define h.bad-empty-read
  {--> number}
  -> (h.newest (h.empty)))
```

Removing the only item leaves no newest item:

<!-- reject: h.bad-after-remove -->
```shen
(define h.bad-after-remove
  {--> number}
  -> (h.newest (h.remove (h.append 7 (h.empty)))))
```

A dishonest implementation cannot claim growth:

<!-- reject: h.bad-growth -->
```shen
(define h.bad-growth
  {A --> (h.history N A) --> (h.history (hs N) A)}
  X H -> H)
```

The replacement signature fixes the length index; its checked body reconstructs
one cell with the original predecessor. The index alone does not establish that
it doubles the value or preserves the *same* predecessor: the body and assertions
supply that evidence. Persistence follows from immutable list construction and
is exercised by retaining the old values. No mutation, operation declarations,
or untyped boundary is used. These are open datatype rules: the claims assume no
later hostile extensions or redefinitions. Indices alone do not prove ordering,
purity, or termination of arbitrary future operations.
