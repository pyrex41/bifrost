# Vocabulary-driven phrases

The vocabulary is data, not a table of answers. The generator takes its color list as an argument so extension changes only the source supplied to it. Empty strings represent epsilon; joining handles either side without introducing spaces.

```shen
(define v-join {string --> string --> string}
 "" Y -> Y
 X "" -> X
 X Y -> (cn X (cn " " Y)))
(define v-row {string --> (list string) --> (list string)}
 _ [] -> []
 C [O | Os] -> [(v-join C O) | (v-row C Os)])
(define phrases {(list string) --> (list string) --> (list string)}
 [] _ -> []
 [C | Cs] Os -> (append (v-row C Os) (phrases Cs Os)))
(define v-colors {--> (list string)} -> ["amber" "violet"])
(define v-extended {--> (list string)} -> (append (v-colors) ["teal"]))
(define v-objects {--> (list string)} -> ["kite" "cup"])
(define unique? {(list string) --> boolean}
 [] -> true
 [X | Xs] -> (and (not (element? X Xs)) (unique? Xs)))
(define vcheck {string --> boolean --> boolean}
 _ true -> true
 L false -> (simple-error L))
(vcheck "four" (= (phrases (v-colors) (v-objects)) ["amber kite" "amber cup" "violet kite" "violet cup"]))
(vcheck "six" (= (phrases (v-extended) (v-objects)) ["amber kite" "amber cup" "violet kite" "violet cup" "teal kite" "teal cup"]))
(vcheck "unique" (and (unique? (phrases (v-colors) (v-objects))) (unique? (phrases (v-extended) (v-objects)))))
(vcheck "epsilon" (= (phrases ["" "amber"] ["" "cup"]) ["" "cup" "amber" "amber cup"]))
```

Each color generates one row over all objects; concatenating all rows gives the Cartesian product, so every pair occurs exactly once if the input lists have unique entries. Rendered strings are unique only if distinct pairs render distinctly, which is true for these words but not arbitrary strings or epsilon-containing vocabularies. The explicit uniqueness checks cover the two supplied vocabularies.
