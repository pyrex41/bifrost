# Trial report

Final checker result: 10 blocks passed, 0 failed. This counts fenced code blocks, not completion of every requested task. In particular the parser remains incomplete.

- **vocabulary:** Checked typed Cartesian product over source lists; exact four- and six-item output, uniqueness, and empty-word spacing assertions pass. Completeness assumes finite source lists; uniqueness is not guaranteed for arbitrary ambiguous vocabularies. No trusted declarations.
- **history:** Checked structural indexed list and implementation bodies; assertions cover newest, persistence, predecessor, and doubling. Three expected type errors cover empty read, read after removing the sole item, and false growth. No operation axioms. Indices do not prove arbitrary behavioral properties.
- **prerequisites:** Untyped Prolog facts and visited-path search, with a trusted `raw-requirements` declaration; typed deduplication and tests for the three jobs pass. Ground input and finite graph are essential; membership negation is applied after the edge has instantiated the node. The trusted declaration and Prolog implementation are not verified by the type checker.
- **parser:** Incomplete. Only a checked expression datatype and evaluator with tree-evaluation assertions are supplied; no token parser, malformed input tests, complete-consumption check or checked parser actions. This is not a solution to the parsing requirement.
- **transition:** Checked configured pure-looking transition with two fees, replay, credit and overdraft assertions; caller must supply nonnegative amounts and fee. Untyped private mutable vector closure tests alias sharing and independent factory allocation. Behavioral tests and types do not prove purity, noninterference or general determinism.
- **review:** Prose counterexample and repair strategy; no execution claimed.

Commands actually run (the checker command was run four times):

```sh
ls input-skills; echo "$BIFROST_TRIAL_BIN"; find input-skills -type f
mkdir -p answers/{vocabulary,history,prerequisites,parser,transition,review}
$BIFROST_TRIAL_BIN skills check --impl shen-jit --dir answers
$BIFROST_TRIAL_BIN skills check --impl shen-jit --dir answers
rg -n 'absvector|vector->|<-vector|\(vector ' input-skills "$BIFROST_PORTS_ROOT" -g '*.shen' | head -35
$BIFROST_TRIAL_BIN skills check --impl shen-jit --dir answers
$BIFROST_TRIAL_BIN skills check --impl shen-jit --dir answers
```

The first checker run failed on `same-set` (an unsuitable `all` application) and on the reserved system name `step`. Replaced these with an explicitly recursive subset check and `account-step`; subsequent runs passed. The final run includes the partial parser evaluator. Files were inspected and written using the harness read/write/edit tools, not shell commands.
