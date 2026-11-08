---
name: shen-review
description: Checklist for reviewing Shen code, your own or someone else's, for the habits that waste the language — hard-coded outputs instead of rules, idle types, hand-written search and parsing, ceremony types, complected interpreters — and for type-system mistakes. Use after writing Shen, before calling Shen work done, or when asked to review or improve a Shen program.
---

# Reviewing Shen

Run this after writing Shen and before calling it done. Each smell names the fix and the skill that covers
it. A smell is a question to answer, not an automatic defect: a table can be the specification, and an
untyped helper can be the right boundary. But every one you keep needs a reason you could state.

## 1. The rules are missing

- **Output written as data.** Finished domain strings, or values some rule could derive, written out by
  hand. Count: if adding one rule means adding N clauses or rows, the rules are missing. Write the rules
  and derive the output (`shen-thinking`).
- **Literal strings that encode a rule.** `"an owl"` written as text means the a/an rule is not in the code.
  Look for lists of near-duplicate strings.
- **The same fact stored twice** (a course list beside the prerequisite table). Derive one from the other.
- **Strings in the middle of the pipeline.** Represent with symbols, lists and trees; render once, at the
  edge.
- **A program longer than its problem statement, with no rules or interpreter in it.** Look for the
  language hiding in the problem.

## 2. Shen's engines are bypassed

- **A hand-written search** with a worklist, a visited set or explicit backtracking over facts →
  `defprolog` (`shen-logic-grammars`).
- **An `if`/`cases` ladder or an association list where each branch is one fact** → Prolog facts,
  `datatype` rules or `defcc` alternatives.
- **A hand-written tokenizer or parser**, especially index arithmetic over a string → `defcc` over a list
  of tokens, or of character codes as in TBoS 22.2.
- **An interpreter written over your own rule format when the rules are a relation or a grammar.** Shen
  already ships that interpreter.

## 3. The types are idle

- **Signatures that say only `symbol`, `string`, `number` or `(list symbol)`** where the domain has a
  notion of valid. Ask what makes a value valid, and whether that belongs in a `datatype` rule
  (`shen-types`).
- **A runtime check nothing depends on.** If a validity test exists, make it a `where` guard that
  discharges a `: verified` premise, so leaving it out becomes a type error.
- **No negative case.** Break one rule on purpose. If the checker still accepts the program, the rule
  isn't in the types.
- **Loaded untyped for no reason.** Every `define` should have a signature unless it is Prolog, a macro,
  a `defcc` without a type, or a deliberate boundary.

## 4. The types are ceremony

- **Wrapper tags that carry no information** (`[qty "one"]`, `[drink "tea"]`) added only so a datatype
  has something to match. The list already had a shape: describe it.
- **Domain vocabulary frozen into types**, as one axiom per literal or an `element?` side condition over
  words that are really data the program reads. Keep changing vocabulary in data; type its shape.
- **A refinement type on internal plumbing** that never meets untrusted data. Type invariants at
  boundaries, shapes everywhere.
- **The same invariant checked in many functions.** The opposite smell: a boundary type is missing.
- **A closure-based abstract type around data nobody needs to hide.** Opaque values can't be printed,
  compared or stored.

## 5. Things complected together

- **Domain words in the interpreter.** A function that both walks a structure and mentions `"tea"`.
  Walking and vocabulary are two things.
- **Global mutable state** (`set`, or a vector used as a database) holding rules that could be an
  argument. Values passed in can be tested; places must be set up first.
- **A macro that expands into a table of data**, or one a function could replace. Macros are for syntax.
- **An `else` branch used as a type test** under an open datatype. Test positively (`string?`,
  `symbol?`) and fail loudly on anything else.

## 6. Type-system mistakes

Check these when the checker rejects, loops or seems to prove too much (details in `shen-types`):

- A single-line rule where a double one was meant, so the body learns nothing about the parts.
- A side condition on a constructor's variable, or a partial one (`(> N 0)` without `number?`).
- A side condition calling a function defined in the same file.
- Circular subtyping, or a left rule that puts its hypothesis back: `maximum inferences exceeded`.
- A function name used as a tag (`[+ X Y]`); `hd`/`tl` instead of `head`/`tail`; a mixed list literal.
- Expecting GADT-style refinement of a type index inside a clause.
- `defprolog` in a typed file; `(tc +)` inside the file it should govern.
- An `if` expected to refine a type (only `where` does, unless `refine-if` is in force).
- Rules that prove everything: a conclusion with no premises and a free type. Every axiom over untyped
  code, and every `declare`, is a trust boundary. Are they few, together and commented?

## 7. Proof of work

- The file loads with the checker turned on *before* the load:
  `bifrost eval -e '(do (tc +) (load "prog.shen"))'` on any port. A `(tc +)` inside the file, or running
  it as a script, checks nothing, so "it ran" is not "it typechecked".
- The file ends with small examples whose answers you predicted by hand, and they match.
- At least one deliberately wrong use is rejected by the checker, if the program claims to enforce a
  rule.
- Watch `typechecked in N inferences`: a sudden jump means overlapping or non-disjoint rules.
