# Shen agent skills

Skills that teach coding agents to write Shen that uses the language: its sequent-calculus types, Prolog,
typed grammars and macros, rather than Lisp or ML written in Shen syntax. Each skill is a folder with a
`SKILL.md` in the [Agent Skills](https://agentskills.io) format: YAML front matter (`name`, `description`)
and plain Markdown. Harnesses that don't read skills can include the Markdown as it is.

| Skill | What it does |
|---|---|
| [`shen-thinking`](shen-thinking/SKILL.md) | The entry point: rules over outputs, choosing an engine, where types go, how to prove it |
| [`shen-types`](shen-types/SKILL.md) | The sequent-calculus cookbook and its pitfalls |
| [`shen-logic-grammars`](shen-logic-grammars/SKILL.md) | `defprolog`, `defcc`, rules as data with an interpreter, macros, and typed boundaries for each |
| [`shen-review`](shen-review/SKILL.md) | The checklist to run before calling Shen work done |

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

An installed skill is not overwritten unless you pass `--force`, which replaces it whole. Name skills after
the flags to install only those.

## Every example is checked

The complaint these skills answer is plausible Shen that dodges the type system, so their examples are not
trusted either. `bifrost skills check [--impl PORT] [--dir skills]` runs every `shen` code block on a real
port under `(tc +)`:

- The blocks of one Markdown file build on each other, like a literate program. Each is loaded as a file
  of its own, after the earlier ones, because a datatype is compiled when its file is read.
- An HTML comment on the line before a fence changes what is expected:
  - `<!-- reject -->`: the checker must refuse the block with a type error. These show a rule being
    enforced.
  - `<!-- untyped -->`: loaded under `(tc -)`, as `defprolog`, macros and a `defcc` without a type must
    be. Later blocks are typechecked again.
  - `<!-- skip -->`: not checked (a fragment, or the bad example of a pair).
  - `<!-- fresh -->`: checked, but without the earlier blocks.

CI runs it on Shen/JIT. Run it on any port you change these skills for.

## Where they come from

The skills were drafted from a review of the programs that accompany Mark Tarver's *The Book of Shen*,
4th edition (<https://shenlanguage.org/TBoS/index.html>), by three reviewers each writing in the manner of
a well-known programmer. None of those people wrote or endorsed this text:

- **Tarver's lens**: a type is a set of objects defined by rules; write the specification as datatypes and
  let the checker prove the program obeys it.
- **Peter Norvig's lens**: problem, then representation, then rules; pick the engine the rules are already
  written in, and build the small interpreter that makes the solution obvious.
- **Rich Hickey's lens**: put the rules in a value; Shen's datatypes are open predicates over existing data,
  closer to a spec than to ML types; type shapes everywhere, invariants only at boundaries.

They disagreed about how much to type. The skills keep the disagreement and resolve it by the problem:
fixed rules you want proved go in datatypes; rules that change, are many or arrive at run time go in data
with a boundary type; relations and grammars go to Prolog and `defcc`. In every case the rules appear
once and the output is derived.

The skills cite the book's sections and use their own examples; none of the book's code is copied here.
The reviews also checked those programs on kernel 43.4. None is broken by the kernel; a few need files
or definitions from other sections, and two (10.1 and 10.3) show the macro pitfalls described in
`shen-logic-grammars`.

## Adding to them

Keep each skill to one job, cite TBoS sections for the patterns, and make every claim about the checker a
block that `bifrost skills check` runs, with a `reject` block where you say something is refused.
