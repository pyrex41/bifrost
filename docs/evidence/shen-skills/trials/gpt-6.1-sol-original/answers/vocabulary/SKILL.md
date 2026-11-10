# Vocabulary-driven generation

The source vocabulary contains alternatives of token sequences. The generic product
chooses one alternative per slot and concatenates tokens; rendering happens last.
An empty alternative is `[]`, not a token containing a space.

```shen
(define v.colors
  {--> (list (list string))}
  -> [["amber"] ["violet"]])

(define v.objects
  {--> (list (list string))}
  -> [["kite"] ["cup"]])

(define v.extended-colors
  {--> (list (list string))}
  -> (append (v.colors) [["teal"]]))

(define v.product
  {(list (list (list string))) --> (list (list string))}
  [] -> [[]]
  [Choices | Slots] ->
    (let Rest (v.product Slots)
      (mapcan (/. Choice (map (/. Words (append Choice Words)) Rest)) Choices)))

(define v.render
  {(list string) --> string}
  [] -> ""
  [Word] -> Word
  [Word | Words] -> (@s Word " " (v.render Words)))

(define v.phrases
  {(list (list string)) --> (list string)}
  Colors -> (map (fn v.render) (v.product [Colors (v.objects)])))

(define v.distinct?
  {(list A) --> boolean}
  [] -> true
  [X | Xs] -> (and (not (element? X Xs)) (v.distinct? Xs)))

(define v.assert
  {boolean --> boolean}
  true -> true
  false -> (error "vocabulary assertion failed"))

(v.assert (= (v.phrases (v.colors))
             ["amber kite" "amber cup" "violet kite" "violet cup"]))
(v.assert (v.distinct? (v.phrases (v.colors))))
(v.assert (= (v.phrases (v.extended-colors))
             ["amber kite" "amber cup" "violet kite" "violet cup" "teal kite" "teal cup"]))
(v.assert (v.distinct? (v.phrases (v.extended-colors))))
(v.assert (= (map (fn v.render) (v.product [[[] ["amber"]] [["kite"]]]))
             ["kite" "amber kite"]))
(v.assert (= (map (fn v.render) (v.product [[["amber"]] [[]]])) ["amber"]))
(v.assert (= (map (fn v.render) (v.product [[[]] [[]]])) [""]))
(v.assert (= (v.product [[]]) []))
```

The extension changes only source vocabulary: it appends the `teal` alternative;
no derived phrase is added to the generator. Expected outputs above are test
oracles, not implementation data. By induction on slots, the base product contains
the one empty choice, and the step enumerates every choice paired with every
suffix. Thus the two finite slots yield 2×2 and 3×2 results. Uniqueness assumes
distinct alternatives and unambiguous rendering (here single, nonempty words with
no embedded spaces). The generic product does not promise deduplication for
arbitrary vocabularies. Empty alternatives skip a slot; empty *strings* inside a
nonempty token list are outside the spacing convention. Shape types do not prove
this completeness or uniqueness argument. No trusted declarations are used.
