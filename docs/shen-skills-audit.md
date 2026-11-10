# Shen skills refinement: integrated audit

## Outcome and provenance

Implemented and cross-reviewed a refinement of `6b04a8387c175c738ca06da7dddf4631abe9ab44`.
Fetched `origin/main` at `51a8f0d1a0eb61be07451d3a64700d4c7ddfa784`, created branch
`refine/shen-skills-evidence` in a separate worktree, and cherry-picked the candidate as
`183e71e041e91740dc33c828f674823dd9815fcc`. Upstream kernel-43.4 adapter updates are retained.
The original `bifrost` main worktree and `shen-behavior` were not modified. At the time of this
audit, no push or remote CI verification had been performed.

User-requested **SOL-6.1** implemented checker hardening; **TERRA-6.1** refined content.
They then independently cross-reviewed the opposite workstream. Fresh model sessions ran and reviewed
the transfer trials. The default subagent launcher failed for missing Anthropic credentials and the
`openai` shared-subscription endpoint hit its limit; work actually ran through explicit
`openai-codex/gpt-6.1-sol` and `openai-codex/gpt-6.1-terra` CLI sessions.
No alternative model's work is presented as those requested models.

## Plan disposition

| Stage | Disposition and evidence |
|---|---|
| Baseline audit | Reproduced **57/57** candidate blocks on JIT and Scheme, both kernel 43.4. The historical report was not reused as current evidence. Original pasted attachment was not recovered; scope is the agreed plan and available source. |
| Checker contract | Hardened intended-diagnostic matching, missing-helper checks, typed load boundaries, annotation parsing and marker handling. Process failure/timeout, rejected-state isolation, fresh blocks, recursive references and exact packaging bytes have regressions. **44/44** real-port adversarial fixtures pass. |
| Concrete correctness findings | All twelve requested topics have a disposition and executable coverage in the [content claim ledger](shen-skills-content-audit.md). Most were already corrected in `6b04a83`; preserved rather than rewritten. |
| Teaching progression | Existing short entries and linked references retained; common problem → judgment → assumptions → rules → implementation → evidence → limitations structure made explicit. Concrete indexed composition explains premise availability. |
| Flagship examples | Added preparation-timing observation, checked top transformation with retained history, empty-after-pop rejection and abstract representation/equality observation. General recursive indexed map was not successfully established; the documented top-only transform is intentionally narrower. |
| Source review | Displayed headings checked selectively. Corrected mode and macro-heading links and removed unsupported section-number history. Some broad references remain unverified; see independent content review. |
| Transfer evaluation | Four fresh six-task batteries and two artifact/protocol reviews completed. One baseline excluded for external-source access. **No reproducible or causal improvement established.** See [trial report](shen-skills-trials/README.md). |
| Packaging/integration | Go tests/vet/build; source, embedded and raw-installed checks; byte comparisons; local links and explicit untyped-boundary inventory. No remote integration or all-port qualification claimed. |

## Claim-to-evidence map

| Claim | Location | Evidence class / scope | Status |
|---|---|---|---|
| A green negative targets the requested diagnostic | `skills_check.go`, `TestJudgeCheck`, `TestSkillsCheckerOnPorts` | Synthetic and real-runtime counterexamples, relative to diagnostics and arity metadata | Corrected; not a security boundary or complete dependency proof |
| Setup failures, process failures and timeout cannot count as rejection | Same checker/tests | Actual failed/timed-out subprocesses and setup fixtures | Confirmed on tested processes/ports |
| Examples survive recursive discovery and installation | `skills_test.go` packaging/accumulation cases | Complete source/embedded/installed byte comparisons and installed execution | Confirmed for tested snapshot |
| Unit restrictions, guards, mixed tokens, callable symbols | `shen-types/references/refinements.md`, typed grammars | Valid controls, named rejections and behavior relative to active theory | Existing corrections retained |
| Ground search, cycles, isolated catalog members and extension behavior | Prolog/generation references | Finite behavioral cases plus stated operational arguments | Qualified: not general completeness/termination proofs |
| Closure state/preparation differs from explicit transitions | `closures-and-state.md` | Aliasing, independence, preparation timing and replay observations | Confirmed sampled behavior; arrow types do not prove effects |
| Structural stack bodies preserve length judgments | `indexed-stacks.md` | Checked implementations, premise derivation, negative bodies/clients | Confirmed relative to rules; content laws separately tested |
| Representation equality is not semantic equality | `invariants-and-abstraction.md` | Equal-size, differently ordered bag observation | Confirmed observation; no multiset-equality implementation claimed |
| Revised skills improve agent transfer | Trial artifacts/reviews | One paired SOL sample; TERRA baseline contaminated | **Unresolved** |
| All citations are independently verified | Content/source review | Selective page-image inspection, not full source audit | **Not claimed** |

Detailed per-correction locations, assumptions and source checks are in the
[content audit](shen-skills-content-audit.md); checker changes and intermediate failures are in the
[checker audit](shen-skills-checker-audit.md). Independent reports:
[content](evidence/shen-skills/content-review.md), [checker](evidence/shen-skills/checker-review.md).

## Fresh integrated verification

Parent reran Go tests/vet/build after concurrent edits settled, then the optional real-port suite:
`BIFROST_SKILLS_TEST_IMPLS=shen-jit,shen-scheme go test -count=1 -run '^TestSkillsCheckerOnPorts$' -v`.
All **22 fixtures per port** passed. Final parent checks of source, embedded and raw-installed content:
**61 blocks, zero failed** in each of six runs (three forms × two ports). Go test/vet/build and
all 61 skill-local links passed. See the [final command output](evidence/shen-skills/final-verification.log).

JIT was explicitly selected using the official v0.5 macOS-arm64 release binary:
kernel 43.4/native 0.5; SHA-256
`949a221bbc4fd8224fff83b394266e2717bffc7b272ce7a4ca542e1df144633c`.
The release archive checksum is separately recorded in the content audit.
Scheme was the installed local 43.4 / Scheme 0.54 / Chez 10.3.0 binary, not a freshly rebuilt
or independently source-reproduced release. The earlier CL 43.1 result is additional attributed
evidence, not mislabelled as 43.4.

Inventory: **33 typed accepted, 22 named rejections, 6 explicitly untyped, zero skipped Shen blocks**.
The six untyped boundaries are explained in the independent content review. All 61 local links
within the four skills were verified by that review and rerun by the parent at final integration. Final integrated commands/logs are archived
under [evidence/shen-skills](evidence/shen-skills/).

## Remaining obligations

- Repeat clean, isolated transfer trials before claiming an improvement; the acceptance criterion
  about reducing agent failure modes is still open.
- Finish source verification if requiring every broad/book/external citation to be independently checked.
- Keep checker limitations visible: arity/diagnostic trust, conservative expanded-form traversal,
  repeated read-time macro effects, open datatype theories, and no hostile-code isolation.
- Some parser/macro/search policies are documented rather than exhaustively tested; no semantic
  soundness, representation-independence, performance or all-port theorem follows from this suite.
- Review the candidate before merging; local verification does not establish remote CI success.
