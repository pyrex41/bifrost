---
name: shen-thinking
description: How to design and write Shen programs that use the language's power (sequent-calculus types, Prolog, typed grammars, macros) instead of Lisp or ML written in Shen syntax. Use before writing or changing any Shen code (.shen files, Shen REPL work, ShenOS kernel or port code), and whenever a task gives rules, constraints or a grammar that a Shen program must follow.
---

# Thinking in Shen

Shen can express every pattern of the functional languages you know, so it is easy to write Lisp or ML in
Shen syntax and stop there. Don't. What Shen has that they lack is a type system the programmer extends with
inference rules, a Prolog that shares the type checker's engine, and typed grammars. A good Shen program
states the problem's rules once, in one of those, and lets Shen enforce or run them.

The failure this skill exists to prevent: asked for a program that generates strings from a ruleset, a
model wrote a giant case statement of finished strings. It passed the test and contained no rules. The
rules had been run once, in the model's head, and only their output was kept. Nothing could check it,
extend it or explain it.

## The one rule

**Write down the rules, not what they produce.** If the output can be derived from rules, the program must
contain the rules. A table of results is only acceptable when the table *is* the specification (a
lookup of facts nobody derives), never when it is a cached run of rules you could have stated.

## Workflow

1. **State the rules in prose first.** List the categories (what kinds of thing exist), the rules that
   combine them, and the facts they depend on. Write each down once. If you can't, ask before coding.
2. **Pick the engine that already has the rules' shape.** Shen has several rule languages. Most of the
   skill is noticing which one the problem is written in:

   | The rules say... | Use | Skill |
   |---|---|---|
   | what a value *is*: well-formed, valid, allowed, agreeing | `datatype` sequent rules | `shen-types` |
   | who is related to whom; constraints; search with backtracking | `defprolog` | `shen-logic-grammars` |
   | which token sequences are allowed, and what they mean | typed `defcc` | `shen-logic-grammars` |
   | how terms evaluate, rewrite or generate, with rules that change or arrive as data | a ruleset value plus a small typed interpreter | `shen-logic-grammars` |
   | a fixed, small case analysis on a value's shape | `define` with patterns and `where` | below |
   | repetitive *syntax* with a fixed meaning | `defmacro` | `shen-logic-grammars` |
   | an infinite or partly consumed sequence | `freeze`/`thaw` (lazy) | |

3. **Represent with structure, render at the edge.** Build lists of symbols and trees the rules can talk
   about. Turn them into strings with one generic walk at the very end. Strings in the middle of a
   pipeline are a sign the rules are missing.
4. **Type it, in proportion.** Under `(tc +)`:
   - Give every `define` a signature. It is cheap, and it documents.
   - Describe the *shape* of data that recurs with a `datatype`: one double-line rule per shape, over the
     lists you already have. No wrapper tags.
   - Put an *invariant* in a type where it pays: where the rules are fixed and you want the checker to
     prove that every function obeys them, or where data crosses a trust boundary (a file, a user,
     another module) and many consumers would otherwise check it again. A runtime test becomes a type
     through a `verified` rule and a `where` guard (see `shen-types`).
   - Don't make a refinement type for every adjective, and don't freeze domain vocabulary that changes into
     types. If the words or rules are data the program reads, keep them in data and type their shape.
5. **Prove it.** Load the file with the checker already on (see "Checking your code": `(tc +)` inside
   the file checks nothing). Add a negative case: a deliberately wrong use that the
   checker must reject. If breaking a rule doesn't produce a type error, the rule isn't in the types.
6. **End with small, predictable examples.** A few top-level expressions whose answers you can work out by
   hand ("24 sentences", "ada may take cs220"). Under `(tc +)` each also prints its type.
7. **Review against `shen-review`** before you call it done.

### Types or data? Resolving the tension

Two good instincts pull apart here. One says *types are specifications*: put the rules in datatypes and the
checker proves the program obeys them. The other says *rules are data*: keep them in plain values an
interpreter reads, and keep types to shapes and boundaries. Both are right, about different problems:

- The rules are **fixed** for the program and you want a **proof** that generated or accepted values obey
  them → datatypes. Breaking a rule becomes a type error.
- The rules **change**, are **many**, or arrive **at run time** → a ruleset value and a small interpreter,
  with one boundary type (`verified`) that says the ruleset was checked.
- The rules are a **relation** or a **grammar** → Shen already ships the interpreter: `defprolog` or
  `defcc`. Don't write your own.

In every case the rules appear exactly once in the program, and the output is derived.

## Worked example

The task: *generate the sentences of a tiny English fragment. Determiner, noun and verb agree in number;
"the" goes with either.*

What a model typically writes (don't):

<!-- skip -->
```shen
(define sentences
  {--> (list string)}
  -> ["a cat sleeps." "a dog barks." "every cat sleeps." "every dogs bark." ...])
```

It typechecks, and so does its bug: `"every dogs bark."` breaks the agreement rule. The types say "strings"
and nothing more, because the rules were never written down.

The Shen way: the lexicon is a family of types indexed by number, and agreement is one shared variable.

```shen
(datatype lexicon
  if (element? W [a every])      ____ W : (det sg);
  if (element? W [some all])     ____ W : (det pl);
  if (= W the)                   ____ W : (det N);
  if (element? W [cat dog])      ____ W : (noun sg);
  if (element? W [cats dogs])    ____ W : (noun pl);
  if (element? W [sleeps barks]) ____ W : (verb sg);
  if (element? W [sleep bark])   ____ W : (verb pl);)

(datatype sentence
  D : (det N); C : (noun N); V : (verb N);
  ========================================
  [D C V] : sentence;)

\\ The shared N in the signature is the agreement rule.
(define clauses
  {(list (det N)) --> (list (noun N)) --> (list (verb N)) --> (list sentence)}
  Ds Cs Vs -> (mapcan (/. D (mapcan (/. C (map (/. V [D C V]) Vs)) Cs)) Ds))

(define fragment
  {--> (list sentence)}
  -> (append (clauses [a every the] [cat dog] [sleeps barks])
             (clauses [some all the] [cats dogs] [sleep bark])))

(define sentence->string
  {sentence --> string}
  [D C V] -> (make-string "~A ~A ~A." D C V))

(length (fragment))
(sentence->string (head (fragment)))
```

`(length (fragment))` is 24 and the first sentence is `"a cat sleeps."`. Now ask for a sentence that
breaks the rule. The checker refuses it before anything runs:

<!-- reject -->
```shen
(define wrong
  {--> (list sentence)}
  -> (clauses [every] [dogs] [bark]))
```

To extend the language, add a rule, not rows. (This example follows the typed grammar of English in
The Book of Shen, 20.4, which goes further and parses sentences into typed logical formulas.)

## Checking your code

Typecheck a whole file portably, on any port (`bifrost` runs it on the active one):

```
bifrost eval -e '(do (tc +) (load "prog.shen"))'
```

**`(tc +)` written inside a file does not typecheck that file**, and neither does running the file as a
script. `load` reads the setting once, when it starts, so an ill-typed program with `(tc +)` on its first
line loads and runs as if it had been checked. (Verified on shen-jit, shen-go, shen-cl, shen-scheme and
shen-lua.) Turn the checker on *before* loading, as above, or type `(tc +)` in the REPL and then
`(load "prog.shen")`. A type error stops the load with `type error in rule N of F`. Never report code as
typechecked because it ran.

## Reading

- *The Book of Shen*, 4th edition, by Mark Tarver, and its programs, which are organised by section:
  <https://shenlanguage.org/TBoS/index.html>. The type chapters are 19 to 28. These skills cite sections
  rather than copying the book's code.
- The other skills: `shen-types` (the sequent-calculus cookbook), `shen-logic-grammars` (Prolog, `defcc`,
  interpreters and macros) and `shen-review` (the checklist).
