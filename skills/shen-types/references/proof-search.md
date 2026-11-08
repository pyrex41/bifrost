# Control datatype proof search

Sections: [modes and unions](#modes-and-unions), [tagged alternatives](#tagged-alternatives),
[cut](#cut), and [debugging](#debugging). Run this file as its own sequence under `(tc +)`.

Shen's datatype rules are both a logical description and a program for proof search. Logically
reasonable rules can lead the automatic checker into an unproductive search. TBoS
[chapter 20](https://shenlanguage.org/TBoS/258.html) treats the control of that search explicitly.

## Modes and unions

A union admits an existing value through either constituent type. A naïve rule from `X : A` to
`X : (union A B)` can also match a goal whose type is still a variable, recursively generating more
unions. A negative mode restricts that matching. It is a search declaration, not logical negation.

Use the book's mode-controlled introduction and elimination rules:

```shen
(datatype union
  X : A;
  _______________________
  X : (- (union A B));

  X : B;
  _______________________
  X : (- (union A B));

  X : A >> P;
  X : B >> P;
  _____________________________
  X : (- (union A B)) >> P;)

(define swap-union
  {(union A B) --> (union B A)}
  X -> X)

(define mixed-values
  {--> (list (union number symbol))}
  -> [1 two])

(if (= (swap-union 7) 7)
    true
    (simple-error "union identity changed"))

(if (= (mixed-values) [1 two])
    true
    (simple-error "union collection changed"))
```

The left rule requires the goal to follow from **both** possible constituent types. It cannot license
numeric arithmetic merely because one constituent is numeric:

<!-- reject: bump-union -->
```shen
(define bump-union
  {(union number symbol) --> number}
  X -> (+ X 1))
```

The mode means `(union A B)` cannot match a bare unknown type in the same way an unrestricted rule
can. Supply the intended type in a signature, as `mixed-values` does, or with an explicit annotation.
Do not expect the checker to invent this union when inferring an unannotated mixed list. Positive
mode restores unification within its scope; the innermost declaration takes priority.

Source: TBoS [§20.7, Mode Declarations](https://shenlanguage.org/TBoS/268.html), especially
[p.255](https://shenlanguage.org/TBoS/269.html), and its
[union program](https://shenlanguage.org/TBoS/Programs/20.7.txt). The longer
[official set example](https://shenlanguage.org/OSM/Sets.html) combines modes, intersection types,
`(without X)`, and guarded membership to check a relational invariant.

## Tagged alternatives

Use a tagged sum when the representation should identify which alternative was chosen. It serves a
different purpose from the union above: `[left X]` and `[right X]` are new list shapes, while a union
contains the unwrapped `X`.

```shen
(datatype either
  X : A;
  _______________________
  [left X] : (either A B);

  X : B;
  ________________________
  [right X] : (either A B);

  X : A >> P;
  ____________________________
  [left X] : (either A B) >> P;

  X : B >> P;
  _____________________________
  [right X] : (either A B) >> P;)

(define either->string
  {(either number string) --> string}
  [left N] -> (str (+ N 1))
  [right S] -> S)

(if (= (either->string [left 41]) "42")
    true
    (simple-error "tagged-sum elimination changed"))
```

The left rules expose the appropriate component after matching a tag. They replace a compound
hypothesis with simpler information. A rule that reproduces the same hypothesis and goal without
progress can loop, even if its logical reading seems harmless.

## Cut

Place `!` after a premise only when finding its proof is sufficient reason to discard the alternatives.
For disjoint primitive cases, it can prevent unnecessary backtracking:

```shen
(datatype printable
  X : number; !;
  ______________
  X : printable;

  X : string; !;
  ______________
  X : printable;

  X : symbol;
  ______________
  X : printable;)

(define show-all
  {(list printable) --> string}
  [] -> ""
  [X | Xs] -> (@s (make-string "<~A>" X) (show-all Xs)))

(if (= (show-all [1 "two" three]) "<1><two><three>")
    true
    (simple-error "printable collection changed"))
```

Source: TBoS [§20.3, Using Cuts](https://shenlanguage.org/TBoS/262.html). Keep the common type
`printable` distinct from the runtime representation: this is a homogeneous typed list whose elements
have different primitive representations.

## Debugging

1. Reduce to the smallest failing definition and record the goal the checker cannot establish.
2. Inspect the trace before increasing the inference limit; see TBoS
   [§20.2, Using spy](https://shenlanguage.org/TBoS/259.html). Repeated goals suggest a loop rather
   than a missing large search budget.
3. Check rule direction and ordering. Two unrestricted conversion rules in opposite directions can
   keep reproving one another. Prefer the direction that advances the required proof.
4. Distinguish control mechanisms: modes restrict matching, cuts discard alternatives, annotations
   supply type information, and `preclude`/`include` select active datatype theories.
5. Make side conditions total on unexpected syntax and keep overlapping shapes deliberate. Measure
   inference counts before and after changes on the same definitions and runtime.

Do not “repair” a search failure with an axiom whose conclusion has an unrestricted type variable.
That may make every judgment derivable. A successful check against an inconsistent theory provides
no useful guarantee.
