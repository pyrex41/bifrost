# Vocabulary-driven phrases

The vocabulary functions are authoritative; the product is always recomputed. `extended` adds teal only at the source, reusing the original colors. Strings denote words, with `""` denoting an empty alternative. No macro or asserted operation types are needed.

```shen
(define v-check
  {string --> boolean --> boolean}
  _ true -> true
  Label false -> (simple-error Label))

(define v-colors
  {symbol --> (list string)}
  original -> ["amber" "violet"]
  extended -> (append (v-colors original) ["teal"])
  _ -> (simple-error "unknown vocabulary version"))

(define v-objects
  {--> (list string)}
  -> ["kite" "cup"])

(define v-join
  {string --> string --> string}
  "" Y -> Y
  X "" -> X
  X Y -> (@s X " " Y))

(define v-row
  {string --> (list string) --> (list string)}
  _ [] -> []
  X [Y | Ys] -> [(v-join X Y) | (v-row X Ys)])

(define v-product
  {(list string) --> (list string) --> (list string)}
  [] _ -> []
  [X | Xs] Ys -> (append (v-row X Ys) (v-product Xs Ys)))

(define v-phrases
  {symbol --> (list string)}
  Version -> (v-product (v-colors Version) (v-objects)))

(define v-unique?
  {(list A) --> boolean}
  [] -> true
  [X | Xs] -> (and (not (element? X Xs)) (v-unique? Xs)))

(v-check "original four in source order"
  (= (v-phrases original)
     ["amber kite" "amber cup" "violet kite" "violet cup"]))
(v-check "extended six from changed source"
  (= (v-phrases extended)
     ["amber kite" "amber cup" "violet kite" "violet cup" "teal kite" "teal cup"]))
(v-check "original has no duplicates" (v-unique? (v-phrases original)))
(v-check "extended has no duplicates" (v-unique? (v-phrases extended)))
(v-check "empty alternatives on either side have no padding"
  (= (v-product ["" "amber"] ["" "cup"])
     ["" "cup" "amber" "amber cup"]))
(v-check "no alternatives is not epsilon" (= (v-product [] ["cup"]) []))
```

For finite lists, `v-row` visits each object once; `v-product` visits each color once and concatenates exactly those rows. This proves completeness of the Cartesian product by structural induction. With distinct source words and unambiguous single-word nonempty colors/objects, rendering is injective, so each phrase occurs once. Arbitrary multiword or duplicate/empty sources could render distinct choices identically; the generator intentionally preserves such multiplicity rather than silently deduplicating it. Epsilon is an alternative, whereas an empty list offers no alternatives. Types establish string-list shape, not this completeness or uniqueness argument. No user declarations or untyped boundaries.
