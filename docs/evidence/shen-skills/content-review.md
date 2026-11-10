# SOL independent content review

## Scope and disposition

Reviewed `/tmp/bifrost-skills-drive/plan.md`, `docs/shen-skills-content-audit.md`, all 13 Markdown files in `skills/shen-{thinking,types,logic-grammars,review}/`, the original `6b04a83` content, and TERRA's working-tree additions. This is the independent **content** review, not authorship/review of the checker implementation. No delegation, commits or pushes. No checker files, `skills/README.md`, or audit ledgers were edited by this review.

**Disposition:** no blocking executable-content or false-proof defect found in the reviewed examples. Four modest citation/provenance edits made below. The plan is not fully completed: source verification and transfer evaluation remain partial, and some behavioral/operational cases are only documented, not executable regressions.

`git diff 6b04a83 HEAD -- skills/shen-thinking skills/shen-types skills/shen-logic-grammars skills/shen-review` was empty: the committed four-skill content matches the original candidate. Original content independently passes 57 blocks; TERRA's current content independently passes 61. These are fresh runs on the specified binary, not adoption of the historical report.

## Findings and fixes

1. **Heading destination, corrected** — `skills/shen-types/references/proof-search.md`: §20.7's displayed heading is **“Controlling Infinite Loops: Mode Declarations”** on `https://shenlanguage.org/TBoS/267.html` (printed p.253). The site's index points to `268.html`, but that scanned page is already a continuation (p.254). Changed the link to the displayed heading and used its full title. Independently read `267.jpg`, `268.jpg`, `269.jpg`, and `Programs/20.7.txt`; p.255 and the companion file agree with the example's mode-controlled union rules. This resolves the ledger's explicitly unresolved union-heading check; it does not validate all mode behavior universally.
2. **Combined heading destination, corrected** — `skills/shen-logic-grammars/references/macros-and-loading.md`: split the combined 10.8–10.9 citation into **10.8 Macro Management → `132.html`** and **10.9 Macroexpansion and Unpackaging → `133.html`**. Both headings were visually checked in their scanned pages. The old link landed in a continuation of 10.8, not at its heading.
3. **Unsupported source-history statement, qualified** — `skills/shen-types/SKILL.md` and `skills/shen-logic-grammars/SKILL.md` stated that program filenames retain older section numbers. The fetched companion files do not establish that historical claim. Replaced it with the warranted instruction to identify sources by displayed title and exact URL rather than infer a prose section/edition from a numbered filename. No historical renumbering story was invented.
4. **Local links, no fix needed** — programmatically checked all 61 local Markdown link destinations, including heading fragments, in the 13 files: zero missing files or anchors. External citation changes do not alter that count. The new composition subsection is reachable within the checked-list section; its absence as a separate top-level contents entry is not a broken link.
5. **Audit invocation rationale is stale** — the ledger says the previous positional skill selector was unsupported. The current checker supports positional selectors: the actual command with `shen-logic-grammars` passed 17 blocks. TERRA's replacement all-skills command in `shen-types/SKILL.md` is valid, so I did not undo it or edit the ledger/checker. Parent should reconcile the rationale with the integrated checker version.

Only my edits are the citation/provenance changes in the four files in findings 1–3. TERRA's pre-existing edits in thinking, closures, indexed stacks and bags are preserved.

## Claim and premise review

- **`ls-map-top` derivation is valid under the displayed active theory.** Its hypotheses supply `F : A --> A` and `S : list-stack A [succ N]`; checked top/pop supply `A` and `list-stack A N`; application supplies another `A`; checked push restores `[succ N]`. No premise is assumed merely because it is the desired conclusion. The implementation signatures themselves are checked against the concrete list rules. It is explicitly a top-only transform, not a successful general recursive indexed map. The new assertion observes transformed top, unchanged predecessor and still-usable original. The new named rejection observes empty access after popping the sole element.
- **Trust boundaries are honest.** Closure-stack operation axioms are explicitly trusted over untyped bodies, unlike checked list-stack implementation bodies. The discussion does not claim length indices prove equations, ordering, provenance, purity or mutation safety. Bag composition only claims preservation relative to asserted operation types. The bag observation correctly separates order-sensitive list equality from semantic multiset equality; it does not pretend to implement `bag=?`.
- **Closures/preparation:** the snapshot/fresh assertion actually distinguishes construction-time reading from invocation-time reading after a vector write. Factory independence, alias sharing, explicit-state replay and persistent predecessors have executable observations. Replay is conditioned on stable dependencies; ordinary arrows are not advertised as effect or termination proofs.
- **Refinements:** component membership is a syntax-time side condition; arbitrary Shen `unit` is not a physical unit vocabulary. Same-file predicate availability is observed under actual load order. `verified` bridges and function-operation contracts are marked as trusted, and successful normal return is separated from exceptions/termination. Simple typed guards are distinguished from guards whose result needs narrower evidence.
- **Prolog:** the catalog includes isolated courses; eligibility explicitly uses direct prerequisites. Negative goals follow grounding goals. Visited-path search has a finite-ground-graph termination argument, excludes the origin even on cycles, and permits duplicate derivations. It is not advertised as unrestricted cyclic transitive closure or complement enumeration. No unsupported completeness proof found.
- **Parser/generator/macros:** typed semantic values are distinguished from recognition completeness, consumption, associativity, termination and search policy. Depth counts nonterminal expansions along a path, not total work; epsilon, bad budgets, undefined references and duplicate names have assertions. Macro helpers are loaded before expansion clients; no binding-hygiene proof is claimed merely from a private trigger.

## Skipped/untyped inventory and absent cases

Fresh inventory: **33 accepted typed blocks, 22 named rejections, 6 untyped blocks, zero skipped Shen blocks**. Two `text` fences in indexed stacks contain equations, and the `text` transition signature in closures is explicitly a conceptual interface, not runnable Shen. Shell examples are checking instructions, not silently skipped Shen.

All six untyped blocks have a local reason and an identified boundary:

- closure-stack representation (1): deliberate unchecked implementation followed by trusted operation axioms and checked clients;
- bag representation (1): deliberately abstract implementation followed by trusted interface axioms;
- Prolog definitions/wrappers (1): `defprolog` layer plus runtime shape validation and expressly trusted declarations;
- macro helpers/macro/client-generated Prolog (3): source-form transformation and generated untyped relation, loaded in phase order, with the typed caller separate.

These are justified example boundaries, **not** evidence that every such helper is intrinsically impossible to type. No new skipped/untyped workaround was introduced.

Remaining gaps, not concealed as completed plan items:

- No held-out comparative agent trials were performed here; green examples are not transfer evidence.
- No independent second-port run in this review. The content ledger's Common Lisp run is attributed evidence, not my verification; no cross-port universal claim follows from the JIT run.
- The bounded generator documents a no-alternatives production and duplicate-output policy without dedicated assertions for those two policies. Depth/result explosion is documented, not stress-tested or bounded by a work budget. Its vocabulary-extension assertion changes a finite production but does not establish synchronization with a separate type-level lexicon or general enumeration completeness.
- Parser tests cover precedence, associativity, leftover input and malformed cases. The overlapping-alternative/limited-backtracking and nullable-cycle cautions are supported by prose/source inspection, not dedicated adversarial runnable cases here. `grammar-error?` catches any runtime exception; its malformed-input assertions are not parse-diagnostic-specific. They should not be cited as evidence of a particular failure message/cause.
- Macro fixture introduces no new runtime binders, so it does not exercise capture avoidance or evaluation-count/order for an expression-rewriting macro. The checklist states those obligations but does not discharge them generally.
- Source review is selective. Visually read scanned headings/pages for §§13.7–13.9; 17.7 and unit discussion p.204; 19.2 and 19.6; chapter 20, §§20.2, 20.3, 20.7 and 20.8; §§21.1–21.2 and 21.4; §22.2; §24.3; §25.6–25.7, including cuts/negation and modes/dynamic updates on printed pp.354–355. Fetched/read companion files `20.7`, `21.1`, `22.2`, `22.3`, `24.3`, `25.6`, `25.7` by content. Did not execute those downloaded book programs. Did not independently audit all ShenDoc/OSM pages, Let Over Lambda, every broad chapter reference, or the complete algebraic-simplification derivation. Remaining source claims are not promoted to independently confirmed facts.
- Original pasted review was unavailable to this review; scope is the supplied plan and actual candidate files. No inference benchmark, kernel soundness proof or semantic representation-independence proof was performed.

## Exact verification

Working directory: `<HOME>/projects/bifrost-skills-evidence`.

Pinned runtime:

```sh
/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit --version
# 43.4 (port ("native" "0.5") implementation ("shen-jit" "0.5"))
shasum -a 256 /tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit
# 949a221bbc4fd8224fff83b394266e2717bffc7b272ce7a4ca542e1df144633c (binary, not release archive)
BIFROST_SHEN_JIT=/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit go run . eval --impl shen-jit -e '(version)'
# 43.4
```

Original candidate extracted outside the repository; checked with the current integrated checker:

```sh
mkdir -p /tmp/bifrost-skills-drive/sol-content-original
git archive 6b04a83 skills/shen-thinking skills/shen-types skills/shen-logic-grammars skills/shen-review | tar -x -C /tmp/bifrost-skills-drive/sol-content-original
BIFROST_SHEN_JIT=/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit go run . skills check --impl shen-jit --dir /tmp/bifrost-skills-drive/sol-content-original/skills
# 57 blocks, 0 failed
```

Current content and final post-edit verification:

```sh
BIFROST_SHEN_JIT=/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit go run . skills check --dir skills --impl shen-jit shen-logic-grammars
# 17 blocks, 0 failed; positional selector works
set -o pipefail
BIFROST_SHEN_JIT=/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit go run . skills check --impl shen-jit --dir skills | tee /tmp/bifrost-skills-drive/sol-content-final.log
# 61 blocks, 0 failed (also passed before my prose edits)
python3 /tmp/bifrost-skills-drive/sol-content-links.py
# 13 Markdown files; 61 local links; 0 broken
# Shen block inventory: {'accept': 33, 'reject': 22, 'untyped': 6}
git diff --check -- skills/shen-thinking skills/shen-types skills/shen-logic-grammars skills/shen-review
# no output, success
go test ./...
# ok github.com/pyrex41/bifrost 5.808s
go vet ./...
# no output, success
```

Source retrieval used `curl -fLsS` for the index, `269.html` and initial program files, then Python `urllib.request.urlopen(..., timeout=30)` downloads of the page images/programs. All inspected material is retained under `/tmp/bifrost-skills-drive/sol-content-sources/`; source URL is `https://shenlanguage.org/TBoS/` plus the basename, or `Programs/` plus the `.txt` basename. Exact downloaded image set: `132 133 163 164 165 216 218 242 249 258 259 262 266 267 268 269 270 271 274 275 276 282 283 293 326 364 368 369 370` (`.jpg`). Visually inspected with the file-reading tool, not inferred from HTML or filenames. The index itself identifies the fifth edition.

Exploratory failure recorded for completeness: `BIFROST_SHEN_JIT=/tmp/bifrost-skills-drive/jit-v05/shen-jit-0.5-macos-arm64/shen-jit go run . eval --impl shen-jit -e '(shen.shen)'` failed with undefined function `shen.shen`; this guessed introspection call was discarded, and the supported `--version` / `(version)` commands above established the runtime. It was not an example failure or successful verification.

Parent should integrate/retest because checker/other workstream edits coexist in this working tree. The ledger's earlier transient unused-import failures are superseded by this review's successful Go test/vet runs, without changing its historical record.
