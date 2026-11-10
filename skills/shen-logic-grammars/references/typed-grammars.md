# Typed grammars and their execution contract

Read this for a parser with typed semantic actions, operator precedence, or a lexer/parser boundary.
The blocks form one example; load each accepted block in order with `(tc +)` already enabled.

- [Token and result types](#token-and-result-types)
- [A grammar with semantic functions](#a-grammar-with-semantic-functions)
- [Checks and failure cases](#checks-and-failure-cases)
- [Text boundaries and parser behavior](#text-boundaries-and-parser-behavior)

## Token and result types

A list has one element type, but that type can include several representations. Numbers, symbols and
nested token lists can all inhabit `token`. This makes `[2 * [3 + 4]]` typeable; it does not decide how a
particular parser interprets grouping. The parser below consumes a flat stream with `lparen` and
`rparen` tokens.

```shen
(datatype token
  N : number;
  ___________
  N : token;

  S : symbol;
  ___________
  S : token;

  X : (list token);
  _________________
  X : token;)

(datatype expression
  N : number;
  ___________
  N : expression;

  X : expression; Y : expression;
  ==============================
  [add X Y] : expression;

  X : expression; Y : expression;
  ==============================
  [sub X Y] : expression;

  X : expression; Y : expression;
  ==============================
  [mul X Y] : expression;)

(define grammar-check
  {string --> boolean --> symbol}
  _ true -> passed
  Label false -> (error "grammar check failed: ~A" Label))

(define nested-tokens
  {--> (list token)}
  -> [2 * [3 + 4]])

(grammar-check "mixed nested token literal"
  (= (nested-tokens) [2 * [3 + 4]]))
```

Use `add`, `sub` and `mul` as explicit AST tags. A bare function name can still be a symbol: the symbol
`+` and `(fn +)` are different values. Some function-name constructor patterns, including `[+ X Y]`,
encounter checker limitations on the tested kernel; that is not a general ban on function names as data.

## A grammar with semantic functions

The number guard narrows a `token` to `number`. Shen-YACC compiles a guard to `if`, so this example
supplies the corresponding `verified` and `if` rules. A guard whose result already has the required type
does not need this refinement. Read the guard/refinement recipe in `shen-types` before extending these
rules: their logical soundness is part of the trusted type environment.

```shen
(datatype guarded-number
  ______________________________________
  (number? X) : verified >> X : number;)

(datatype grammar-refine-if
  Test : boolean; Test : verified >> X : A; (not Test) : verified >> Y : A;
  _____________________________________________________________________
  (if Test X Y) : A;)

(defcc <expression>
  {(list token) ==> expression}
  <term> <sum-tail> := (<sum-tail> <term>);)

(defcc <sum-tail>
  {(list token) ==> (expression --> expression)}
  + <term> <sum-tail> := (/. Left (<sum-tail> [add Left <term>]));
  - <term> <sum-tail> := (/. Left (<sum-tail> [sub Left <term>]));
  <e> := (/. Left Left);)

(defcc <term>
  {(list token) ==> expression}
  <factor> <product-tail> := (<product-tail> <factor>);)

(defcc <product-tail>
  {(list token) ==> (expression --> expression)}
  * <factor> <product-tail> := (/. Left (<product-tail> [mul Left <factor>]));
  <e> := (/. Left Left);)

(defcc <factor>
  {(list token) ==> expression}
  N := N where (number? N);
  lparen <expression> rparen := <expression>;)

(define calculate
  {expression --> number}
  [add X Y] -> (+ (calculate X) (calculate Y))
  [sub X Y] -> (- (calculate X) (calculate Y))
  [mul X Y] -> (* (calculate X) (calculate Y))
  N -> N where (number? N)
  _ -> (error "unhandled expression"))
```

Each tail returns a function that accepts the expression to its left. Applying that function builds
`[sub [sub 8 2] 1]`, so subtraction associates to the left without left-recursive parsing. Semantic
actions can produce numbers, strings, ASTs, or functions according to the consumer's needs. See
[closures and state](../../shen-thinking/references/closures-and-state.md) for the function-value model.

## Checks and failure cases

Use value assertions as well as a checker rejection. A successful typecheck establishes the declared
semantic result type under the active rules; it does not establish that the parser accepts exactly the
intended language, always terminates, or implements the intended precedence.

```shen
(grammar-check "precedence and grouping"
  (= (calculate
       (compile (fn <expression>) [2 * lparen 3 + 4 rparen - 1]))
     13))

(grammar-check "left-associative AST"
  (= (compile (fn <expression>) [8 - 2 - 1]) [sub [sub 8 2] 1]))

(grammar-check "left-associative value"
  (= (calculate (compile (fn <expression>) [8 - 2 - 1])) 5))

(define grammar-error?
  {(list token) --> boolean}
  Tokens -> (trap-error
              (do (compile (fn <expression>) Tokens) false)
              (/. E true)))

(grammar-check "empty input rejected" (grammar-error? []))
(grammar-check "missing operand rejected" (grammar-error? [2 +]))
(grammar-check "trailing token rejected" (grammar-error? [2 3]))
(grammar-check "unclosed group rejected" (grammar-error? [lparen 2 + 3]))
(grammar-check "unrecognised token rejected" (grammar-error? [unknown]))
```

The failure checks above test rejection at run time; they are not type proofs. This malformed semantic
action must instead fail during typechecking, with `<wrong-meaning>` named in the diagnostic:

<!-- reject: <wrong-meaning> -->
```shen
(defcc <wrong-meaning>
  {(list token) ==> expression}
  N := [add 1];)
```

## Text boundaries and parser behavior

Choose and document the lexer output. A byte/character lexer can itself be typed. If a generic reader or
foreign decoder returns dynamically shaped data, validate that result and isolate any necessary
`declare` at that boundary. A declaration is an unchecked assertion, not evidence that the reader or
parser was typechecked. `read-from-string` follows Shen's reader syntax, including nested forms; it is
not automatically a lexer for an arbitrary surface language.

Before accepting a parser, check these operational properties from TBoS Chapter 13:

| Property | What to account for |
|---|---|
| Consumption | `compile` requires the whole input to be consumed. Test trailing tokens. `<end>` asserts the end; `<!>` consumes the remaining input and returns it. |
| Alternative order | A short successful alternative may leave unwanted input. Order overlapping expansions deliberately; longest-first is often necessary. |
| Backtracking | Shen-YACC retries alternatives of a nonterminal; it does not generally revisit an earlier successful nonterminal within an expansion when a later one fails. |
| Recursion | Eliminate direct/indirect left recursion. Ensure recursive cycles consume input; `<e>` itself consumes nothing. |
| Meaning | Specify associativity and precedence independently of recognisability. Keep semantic actions and guards pure when repeated attempts must be harmless. |
| Scope of types | A result type constrains successful semantic values. It does not prove coverage, completeness, termination, or equivalence to a reference grammar. |

For example, `a; a <as>;` can stop after the first `a`, and a greedy nonterminal can consume tokens
needed by the following one. The book's examples in sections 13.7–13.9 explain these cases; changing a
type annotation does not change this search policy.

## Sources

- TBoS, online fifth edition, [13.7, Consuming the Input](https://shenlanguage.org/TBoS/163.html),
  [13.8, Limited Backtracking](https://shenlanguage.org/TBoS/164.html), and
  [13.9, Left Recursion](https://shenlanguage.org/TBoS/165.html).
- [ShenDoc: Shen-YACC](https://shenlanguage.org/SD/Shen-YACC.html) for typed nonterminals and guards.
- TBoS [typed English-to-logic program](https://shenlanguage.org/TBoS/Programs/22.3.txt) for higher-order
  semantic actions, and [typed character-code program](https://shenlanguage.org/TBoS/Programs/22.2.txt).
- [ShenDoc: symbol semantics](https://shenlanguage.org/SD/SymbolSemantics.html) for symbols versus `fn`.
