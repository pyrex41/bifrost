# Rules as data with bounded generation

Read this when a language's productions arrive as data or generation needs an explicit search budget.
Load the accepted blocks in order with `(tc +)` already enabled.

- [Represent and validate productions](#represent-and-validate-productions)
- [Expand with a decreasing budget](#expand-with-a-decreasing-budget)
- [Assert the contract](#assert-the-contract)

## Represent and validate productions

A string is a terminal word; a symbol refers to a named production. Each production has a name and a
list of alternatives. An empty alternative denotes epsilon. The shape type does not say that names are
unique or references are defined, so check those properties separately.

```shen
(datatype grammar-item
  X : string;
  ___________
  X : grammar-item;

  X : symbol;
  ___________
  X : grammar-item;)

(datatype production
  Name : symbol; Alternatives : (list (list grammar-item));
  ========================================================
  [Name | Alternatives] : production;)

(datatype generator-refinements
  ______________________________________
  (string? X) : verified >> X : string;

  ______________________________________
  (symbol? X) : verified >> X : symbol;)

(define production-names
  {(list production) --> (list symbol)}
  [] -> []
  [[Name | _] | Rest] -> [Name | (production-names Rest)])

(define unique-names?
  {(list symbol) --> boolean}
  [] -> true
  [Name | Names] -> (and (not (element? Name Names)) (unique-names? Names)))

(define item-defined?
  {grammar-item --> (list symbol) --> boolean}
  X _ -> true where (string? X)
  X Names -> (element? X Names) where (symbol? X)
  _ _ -> false)

(define items-defined?
  {(list grammar-item) --> (list symbol) --> boolean}
  [] _ -> true
  [X | Xs] Names -> (and (item-defined? X Names) (items-defined? Xs Names)))

(define alternatives-defined?
  {(list (list grammar-item)) --> (list symbol) --> boolean}
  [] _ -> true
  [Alt | Alts] Names -> (and (items-defined? Alt Names)
                            (alternatives-defined? Alts Names)))

(define productions-closed?
  {(list production) --> (list symbol) --> boolean}
  [] _ -> true
  [[_ | Alts] | Rest] Names -> (and (alternatives-defined? Alts Names)
                                  (productions-closed? Rest Names)))

(define valid-productions?
  {(list production) --> boolean}
  G -> (let Names (production-names G)
         (and (unique-names? Names) (productions-closed? G Names))))

(define natural-depth?
  {number --> boolean}
  D -> (and (integer? D) (>= D 0)))
```

These functions receive data whose **shape** is already typed. Decode and validate the shape of external
data before calling them. A `verified` rule for a custom validator is an additional trusted implication;
it is not created merely by giving the validator a `boolean` result type.

## Expand with a decreasing budget

Validate the budget as a nonnegative integer. A signature containing `number` alone would also admit
negative and fractional budgets; decrementing those until equality with zero is not a termination
argument. The recursive step below also rejects a numeric representation for which subtraction does
not decrease the budget.

The budget counts nonterminal expansions along a path. Terminals need no expansion. Accumulate lists
of words and concatenate those lists, so epsilon is the ordinary empty-list identity. Render after
generation; this also makes a nullable prefix and a nullable suffix behave alike.

```shen
(define production-alternatives
  {symbol --> (list production) --> (list (list grammar-item))}
  Name [[Name | Alts] | _] -> Alts
  Name [_ | Rest] -> (production-alternatives Name Rest)
  Name [] -> (error "undefined production: ~A" Name))

(define expand-words
  {(list production) --> grammar-item --> number --> (list (list string))}
  _ "" _ -> [[]]
  _ X _ -> [[X]] where (string? X)
  _ _ D -> [] where (<= D 0)
  G Name D -> (let Next (- D 1)
                (if (< Next D)
                    (mapcan (/. Alt (expand-sequence G Alt Next))
                            (production-alternatives Name G))
                    (error "depth does not decrease"))) where (symbol? Name)
  _ _ _ -> (error "unsupported grammar item"))

(define expand-sequence
  {(list production) --> (list grammar-item) --> number --> (list (list string))}
  _ [] _ -> [[]]
  G [X | Xs] D -> (word-products (expand-words G X D) (expand-sequence G Xs D)))

(define word-products
  {(list (list string)) --> (list (list string)) --> (list (list string))}
  Heads Tails -> (mapcan (/. H (map (/. T (append H T)) Tails)) Heads))

(define render-words
  {(list string) --> string}
  [] -> ""
  [W] -> W
  [W | Ws] -> (@s W " " (render-words Ws)))

(define generate
  {(list production) --> symbol --> number --> (list string)}
  _ _ D -> (error "depth must be a nonnegative integer") where (not (natural-depth? D))
  G _ _ -> (error "duplicate or undefined production") where (not (valid-productions? G))
  G Start _ -> (error "undefined start symbol") where (not (element? Start (production-names G)))
  G Start D -> (map (fn render-words) (expand-words G Start D)))

(define orders
  {--> (list production)}
  -> [[order [qty drink] [qty size drink]]
      [qty   ["one"] ["two"]]
      [size  ["small"] ["large"]]
      [drink ["tea"] ["coffee"] [drink "with" extra]]
      [extra ["milk"] ["lemon"]]])
```

Call `generate` at the boundary; the expansion helpers assume its validation. An empty string terminal
also denotes epsilon in this example. A production with no alternatives generates nothing. Distinct
derivations may repeat the same string in the result.

With finite productions and a valid decreasing budget, recursive rules have only finitely many
expansions. That does not bound the result count to a practical size: Cartesian products can still be
expensive. Add a result/work budget or use a lazy traversal when the caller only needs some results.

## Assert the contract

```shen
(define generator-check
  {string --> boolean --> symbol}
  _ true -> passed
  Label false -> (error "generator check failed: ~A" Label))

(generator-check "recursive grammar count" (= (length (generate (orders) order 3)) 36))
(generator-check "first order" (= (head (generate (orders) order 3)) "one tea"))
(generator-check "new drink alternative"
  (= (generate [[drink ["tea"] ["coffee"] ["water"]]] drink 1)
     ["tea" "coffee" "water"]))
(generator-check "zero budget" (= (generate (orders) order 0) []))
(generator-check "bounded unproductive cycle" (= (generate [[loop [loop]]] loop 3) []))

(generator-check "epsilon prefix"
  (= (generate [[start [optional "tea"]] [optional []]] start 2) ["tea"]))
(generator-check "epsilon suffix"
  (= (generate [[start ["tea" optional]] [optional []]] start 2) ["tea"]))
(generator-check "epsilon alone" (= (generate [[start []]] start 1) [""]))
(generator-check "empty string terminal"
  (= (generate [[start ["" "tea"]]] start 1) ["tea"]))

(generator-check "negative budget rejected"
  (trap-error (do (generate (orders) order -1) false)
    (/. E (= (error-to-string E) "depth must be a nonnegative integer"))))
(generator-check "fractional budget rejected"
  (trap-error (do (generate (orders) order 0.5) false)
    (/. E (= (error-to-string E) "depth must be a nonnegative integer"))))
(generator-check "undefined reference rejected"
  (trap-error (do (generate [[start [missing]]] start 2) false)
    (/. E (= (error-to-string E) "duplicate or undefined production"))))
(generator-check "duplicate name rejected"
  (trap-error (do (generate [[start ["tea"]] [start ["coffee"]]] start 1) false)
    (/. E (= (error-to-string E) "duplicate or undefined production"))))
(generator-check "unknown start rejected"
  (trap-error (do (generate (orders) missing 3) false)
    (/. E (= (error-to-string E) "undefined start symbol"))))
```

A malformed production shape fails earlier, during checking:

<!-- reject: wrong-production -->
```shen
(define wrong-production
  {--> (list production)}
  -> [["start" ["tea"]]])
```

For an API with many calls, validate once and choose an explicit abstraction for the validated grammar
and budget. Consider an opaque package interface or a carefully justified boundary type; preserve the
same runtime checks and make invalid construction paths explicit.

## Sources

TBoS's [SECD program](https://shenlanguage.org/TBoS/Programs/24.3.txt) and
[typed Horn-clause interpreter](https://shenlanguage.org/TBoS/Programs/25.7.txt) are larger examples of
typed interpreters over language terms. This generator is a separate worked example, with its own
depth, epsilon, duplicate and error policies. Consult the current
[book index](https://shenlanguage.org/TBoS/index.html) for the edition and chapter titles.
