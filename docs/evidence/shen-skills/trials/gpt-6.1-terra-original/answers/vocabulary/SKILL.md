# Vocabulary-driven phrases

The two source lists are the vocabulary. A Cartesian product combines them; rendering omits the separator when either alternative is empty. Adding a color only changes the source list passed to the generator.

```shen
(define join-words {string --> string --> string}
  "" B -> B
  A "" -> A
  A B -> (@s A " " B))
(define phrases {(list string) --> (list string) --> (list string)}
  Cs Os -> (mapcan (/. C (map (/. O (join-words C O)) Os)) Cs))
(define original-colors {--> (list string)} -> ["amber" "violet"])
(define objects {--> (list string)} -> ["kite" "cup"])
(define extended-colors {--> (list string)} -> (append (original-colors) ["teal"]))
(define unique? {(list A) --> boolean}
  [] -> true
  [X | Xs] -> (and (not (element? X Xs)) (unique? Xs)))
(define check-v {boolean --> boolean} true -> true false -> (error "vocabulary assertion failed"))
(check-v (= (phrases (original-colors) (objects)) ["amber kite" "amber cup" "violet kite" "violet cup"]))
(check-v (= (phrases (extended-colors) (objects)) ["amber kite" "amber cup" "violet kite" "violet cup" "teal kite" "teal cup"]))
(check-v (unique? (phrases (original-colors) (objects))))
(check-v (unique? (phrases (extended-colors) (objects))))
(check-v (= (phrases ["" "amber"] ["cup" ""]) ["cup" "" "amber cup" "amber"]))
```

Each color is paired once with each object by the nested maps, so the result is complete for finite lists. Uniqueness requires distinct source words and unambiguous rendering; the assertions establish it for these particular inputs, not all inputs (especially empty alternatives).
