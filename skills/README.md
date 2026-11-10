# Shen agent skills

Skills for designing, writing and reviewing Shen programs with functional decomposition, closures,
sequent-calculus types, Prolog, typed grammars and macros. Each skill is a folder with a `SKILL.md` in
[Agent Skills](https://agentskills.io) format: YAML front matter (`name`, `description`) and Markdown.
Harnesses that do not read skills can include the Markdown directly.

The entry files provide decisions and workflows. Linked references contain deeper explanations and
executable examples; installations include those references.

| Skill | Role | Selected references |
|---|---|---|
| [`shen-thinking`](shen-thinking/SKILL.md) | Operations and laws, engine choice, representation, state and proof obligations | [Closures and state](shen-thinking/references/closures-and-state.md); [indexed stacks](shen-thinking/references/indexed-stacks.md) |
| [`shen-types`](shen-types/SKILL.md) | Concrete rules, refinements, modes and compositional invariants | [Refinements](shen-types/references/refinements.md); [proof search](shen-types/references/proof-search.md); [invariants and abstraction](shen-types/references/invariants-and-abstraction.md) |
| [`shen-logic-grammars`](shen-logic-grammars/SKILL.md) | Relational search, typed parsing, rule interpreters and macro phases | [Prolog search](shen-logic-grammars/references/prolog-search.md); [typed grammars](shen-logic-grammars/references/typed-grammars.md); [bounded generation](shen-logic-grammars/references/bounded-generation.md); [macros](shen-logic-grammars/references/macros-and-loading.md) |
| [`shen-review`](shen-review/SKILL.md) | Review questions and evidence required for claimed behavior | Covers the other three skills |

## Install

```bash
bifrost skills list
bifrost skills install --target claude            # ~/.claude/skills
bifrost skills install --target all               # Claude Code, Codex, Pi and OpenCode
bifrost skills install --target codex --project . # ./.agents/skills
bifrost skills install --target raw --dir DIR     # the Markdown files only
```

| `--target` | Global | With `--project DIR` |
|---|---|---|
| `claude` | `~/.claude/skills` | `DIR/.claude/skills` |
| `codex` | `~/.agents/skills` | `DIR/.agents/skills` |
| `pi` | `~/.pi/agent/skills` | `DIR/.pi/skills` |
| `opencode` | `~/.config/opencode/skills` | `DIR/.opencode/skills` |
| `raw` | `--dir DIR`, Markdown only | |

An installed skill is not overwritten unless you pass `--force`, which replaces it whole. Name skills
after the flags to install only those. Install the complete set to preserve links between skills.

## Executable examples

```bash
bifrost skills check --impl shen-jit --dir skills
```

The checker runs every `shen` fence in every Markdown file inside the selected skills, including their
references. Each check starts a fresh port process. Earlier accepted blocks in the **same Markdown file**
are loaded first, then the block under test. Files do not implicitly import examples from other files.

Typed blocks load with `(tc +)` enabled before `load`. Earlier definitions in one loaded block can be
used by later definitions and proof-time side conditions in that block. Helpers needed during read-time
macro expansion must already be loaded before reading the file; those are different phases.

A standalone HTML comment immediately before a fence changes its treatment (blank lines are allowed).
Standalone HTML comments outside fences are reserved for these directives: unknown, malformed, stacked,
detached, dangling, or non-`shen`-fence directives are errors, not silently ignored options.

| Annotation | Meaning |
|---|---|
| `<!-- reject: wrong-client -->` | Must produce `type error in rule N of wrong-client`; matching is literal, case-sensitive and bounded by whitespace, so `other-wrong-client` does not match |
| `<!-- reject -->` | Must produce a definition's `type error in rule N of NAME`; retained for examples without a specific target |
| `<!-- untyped -->` | Load deliberately under `(tc -)`; restore `(tc +)` for later blocks |
| `<!-- skip -->` | Do not execute this fragment; explain why it is incomplete or illustrative |
| `<!-- fresh -->` | Start a new accumulation of accepted blocks within this file |

Rejected blocks are never carried into later checks. Setup loads are outside the rejection trap, so a
setup error cannot satisfy an expected rejection. Each typed load re-enables `(tc +)`, including after
untyped setup. A timed-out or unsuccessfully exited port fails the check even if it printed a success
marker. Markers must occupy their own lines (the rejection diagnostic follows its marker on that line).

Named rejection text may also be a whole-token diagnostic phrase such as `rule 2 of wrong-client`.
Prefer the complete definition name. An undefined helper can produce the same diagnostic as an intended
type mismatch, so after a failed negative load the checker re-reads its expanded forms and rejects
unresolved `(fn NAME)` and zero-argument calls using the port's arity metadata. Kernel 43.4's generated
`shen.parse-failure` call is exempt because that working internal function has no published arity.
The additional read can expand macros again: keep negative blocks small, declarative and free of
read-time side effects; prefer helpers in earlier accepted blocks. Same-file typed forward declarations
remain supported. The walk is conservative: it traverses all reader-expanded forms, including forms
following the failed definition, and is not a general quotation-aware dependency analysis. Keep unrelated
forms out of negative blocks. Pair every claimed exclusion with a valid use.
This is a documentation checker, not a sandbox or a proof that every declared helper is implemented;
arity metadata and the kernel's diagnostic format are trusted.

**Acceptance alone does not check a result's value.** Examples assert expected values with `if` or a
small assertion function that raises an error on mismatch. Use such assertions for behavioral laws,
aliasing, parser results and generator extensions. Distinguish sampled tests from universal proofs.

CI runs the suite on its pinned Shen/JIT release. Run on the ports affected by a change; one successful
port run does not establish portability across all implementations. To typecheck a separate file:

```bash
bifrost eval -e '(do (tc +) (load "prog.shen"))'
```

Putting `(tc +)` inside that file or merely running it as a script is not evidence that the load was
checked. State deliberate untyped representation and declaration boundaries explicitly.

## Sources and scope

The main reference is Mark Tarver's [*The Book of Shen*, online fifth edition](https://shenlanguage.org/TBoS/index.html).
The skills use the following parts together:

| Topic | Book location |
|---|---|
| Functional decomposition and higher-order programming | Chapters 6 and 16 |
| Assignment, vectors and state | Chapters 7–8 |
| Macros and packages | Chapter 10 |
| Grammar execution and its limits | Chapter 13 |
| Types, sequents, concrete datatypes and proof control | Chapters 17–20 |
| Abstract operations, laws and algebraic transformations | Chapter 21 |
| Typed grammars and semantic functions | Chapter 22 |
| Prolog and its implementation | Chapters 25 and 27 |

Do not infer a prose section or edition from a numbered companion-program filename. Follow the
displayed chapter or section heading and the linked code together. The [Shen language reference](https://shenlanguage.org/SD/Primitives.html)
supplies details of current language operations. The closure material also draws on Doug Hoyte's
[*Let Over Lambda*, chapter 2](https://letoverlambda.com/index.cl/guest/chap2.html), adapted to Shen's
semantics rather than treating Common Lisp mechanisms as interchangeable.

The design principle is to state useful invariants and explain how operations preserve them. A boundary
check, a checked transformation and an asserted interface perform different jobs. An operation axiom
over untyped code can constrain clients without proving the implementation. Keep that trust visible.

## Adding to the skills

Keep each entry focused on decisions and point directly to deeper references. Give references over 100
lines a table of contents. Use typed executable examples where possible; explain deliberate untyped
blocks and every skipped fragment. Add named negative cases for claimed exclusions, assert the results
that matter, and check extension behavior when the program presents several views of the same facts.

Preserve existing skill names and installation paths. Run the checker and the relevant Go tests after
editing. Record the actual port/kernel and remaining assumptions when reporting a change. To run the
checker's adversarial fixtures on installed ports (ordinary Go tests do not require a Shen runtime):

```bash
BIFROST_SKILLS_TEST_IMPLS=shen-jit,shen-scheme go test -run TestSkillsCheckerOnPorts -v
```

The Go suite also checks nested discovery, process failures after markers, and exact source/embedded/
installed bytes, including reference files. See [the checker audit](../docs/shen-skills-checker-audit.md)
for observed baseline/current results and evidence limits.
