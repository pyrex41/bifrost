# Macros, packages and load phases

Read this when defining new syntax or generating definitions. Prefer a function when ordinary function
application expresses the required behavior; choose a macro when syntax, evaluation order or generated
definitions are the point.

- [Load helpers, then the macro, then clients](#load-helpers-then-the-macro-then-clients)
- [Review the expansion contract](#review-the-expansion-contract)

## Load helpers, then the macro, then clients

Treat the following blocks as separate files. Shen reads and transforms a file before evaluating its
ordinary definitions. A helper used during expansion must already exist at that phase. Merely moving
its definition earlier in the same client file is not a reliable substitute for loading it first.

This example introduces notation for binary facts whose arguments are symbols. It validates that small
syntax contract and uses a package-internal trigger, `family.facts`, to limit accidental matches.

**1. `fact-helpers.shen`: load the expansion helpers.** These helpers manipulate untyped source forms.

<!-- untyped -->
```shen
(package family []

  (define pairs->clauses
    [] -> []
    [A B | Pairs] -> [A B <-- ; | (pairs->clauses Pairs)]
                    where (and (symbol? A) (symbol? B))
    _ -> (error "facts require pairs of symbols"))

  (define make-facts
    _ [] -> (error "facts require at least one pair")
    Rel Pairs -> [defprolog Rel | (pairs->clauses Pairs)] where (symbol? Rel)
    _ _ -> (error "fact relation must be a symbol")))
```

**2. `fact-macro.shen`: load the macro.** Package-first processing qualifies its private pattern head
and helper names. The generated `defprolog` is ordinary Shen syntax.

<!-- untyped -->
```shen
(package family []
  (defmacro facts-macro
    [facts Rel | Pairs] -> (make-facts Rel Pairs)))
```

**3. `family.shen`: read and load clients after the macro is available.** Export the vocabulary intended
for callers; leave the syntax trigger and predicate private to the package's namespace.

<!-- untyped -->
```shen
(package family [children tom bob ann cal]
  (facts parent tom bob bob ann bob cal)

  (define children
    P -> (prolog? (findall C (parent (receive P) C) Cs) (return Cs))))

\\ Trusted interface for this fixed, ground fact fixture; see prolog-search.md.
(declare children [symbol --> [list symbol]])

(if (= (family.facts-macro [family.facts relation a b]) [defprolog relation a b <-- ;])
    passed
    (error "unexpected fact macro expansion"))

(if (trap-error (do (family.pairs->clauses [a b c]) false)
      (/. E (= (error-to-string E) "facts require pairs of symbols")))
    passed
    (error "malformed fact syntax was accepted"))
```

**4. Check the typed caller.** This block switches back to `(tc +)` through the skill checker. When
loading files manually, enable the checker before loading this file.

```shen
(define macro-check
  {string --> boolean --> symbol}
  _ true -> passed
  Label false -> (error "macro check failed: ~A" Label))

(define children-correct?
  {--> boolean}
  -> (let Cs (children bob)
       (and (= (length Cs) 2) (element? ann Cs) (element? cal Cs))))

(macro-check "generated relation" (children-correct?))
```

The declaration does not prove the generated Prolog predicate or its wrapper. For changing or external
facts, validate inputs and outputs as in [the Prolog boundary](prolog-search.md).

## Review the expansion contract

Before keeping a macro, inspect the generated form and check:

| Concern | Rule of use |
|---|---|
| Matching | Use a distinctive syntax head. Broad patterns such as `[X + Y]` can match reader-generated `cons` forms as well as the intended notation. |
| Scope | Package the trigger and helpers; choose exported symbols deliberately. Packages namespace symbols and constrain matching, but are not a security boundary. |
| Termination | Reader macros are repeatedly composed to a fixed point. A rewrite cycle or interaction between macros can prevent reading from finishing. |
| Evaluation | State which argument expressions execute, how many times, and in which order. Macro expansion can intentionally change strict evaluation. |
| Binding | Use fresh names, such as `gensym`, for introduced bindings or generated definitions. Check capture and substitution; distinct trigger names do not ensure hygiene. |
| Phases | Load expansion-time dependencies before reading clients. Turn the checker on before loading the generated typed program. |
| Diagnostics | Reject malformed syntax with a useful error, and inspect expansions with the macro function or `macroexpand`. |

Use `package null` when one syntax form must expand to several definitions. Keep such expansion logic
separate from runtime helpers so its dependencies and generated names can be reviewed. Consult
`undefmacro` when removing an experimental macro from a working image; a lingering macro can affect
later reads even when the source file is no longer being loaded.

## Sources

- TBoS, online fifth edition, [10.1, Macros](https://shenlanguage.org/TBoS/121.html),
  [10.2, Changing the Order of Evaluation](https://shenlanguage.org/TBoS/124.html),
  [10.6, The Null Package and Macros](https://shenlanguage.org/TBoS/129.html),
  [10.8, Macro Management](https://shenlanguage.org/TBoS/132.html), and
  [10.9, Macroexpansion and Unpackaging](https://shenlanguage.org/TBoS/133.html).
- [ShenDoc: reader macros](https://shenlanguage.org/SD/Macros.html) and
  [packages](https://shenlanguage.org/SD/Packages.html).
