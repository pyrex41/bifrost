# Review: client checking is not implementation checking

The claim is false. The operation rules are trusted axioms over untyped closure code. A client proof uses the asserted type of `push`, without inspecting its body. Changing that body to return the original stack leaves the axioms and hence client proofs unchanged.

## Smallest counterexample (proposed, not executed)

Let `e = empty`, and consider `s = push(1, e)`. The asserted rules assign `s : stack number [succ 0]`, so `top(s)` typechecks. The actual push returns `e`; it has not added an item. Thus the claimed length change and law `top(push(1,e)) = 1` are unsupported. The precise runtime symptom of `top(e)` depends on the empty closure implementation; it might error or return some other value. No execution result is claimed here.

A second distinguishing observation, if a nonempty baseline can be constructed independently, is pushing a different value and testing that top changes. Do not rely on the broken push to create that baseline.

## Smallest repair and evidence strategy

1. Fix push to construct a fresh node/closure retaining both the new element and the old stack. Review the dispatch reductions for `top(push(x,s)) = x` and `pop(push(x,s)) = s`. Closure code left untyped still requires a reviewed correspondence to its asserted interface.
2. Prefer a concrete persistent representation with checked constructors and destructors: empty list at index 0 and cons at successor index, with introduction **and elimination** rules. Check the actual bodies. A proposed push claiming a successor while returning its input should then be a named negative implementation test, not merely a rejected client.
3. Enable checking before loading implementations. Test the two equations, an empty/singleton boundary, multiple distinct values, predecessor observations and old aliases after updates. Require type errors for reading empty and reading after removing the sole item. Assertions must error on mismatch. These are proposed checks, not invented outcomes.
4. If closure representation is retained, explicitly list untyped definitions, asserted operation judgments, and the relation between a closure and the abstract sequence it represents. Establish empty correspondence, push preservation and observation agreement. Differential tests against a checked list representation provide supporting evidence, not universal equivalence.

## What each layer establishes

- **Client checking under axioms:** legal compositions relative to those axioms and the active, extensible datatype theory. It does not validate the axioms or implementation.
- **Checked concrete bodies:** representation/index obligations of each normally returning result. The checker still depends on sound concrete rules and the runtime; it does not supply every semantic law automatically.
- **Length indices:** structural size/nonemptiness only, under the representation assumptions. Two equal-length stacks can have different elements or orders. Even replacing top with the wrong value can preserve the length type.
- **Ordering and predecessor preservation:** require content laws and body reasoning, with tests as finite evidence.
- **Persistence and purity:** require absence of destructive interference and effects, or an explicit justified ownership/effect discipline. Hidden mutation can invalidate an alias's old index. Persistent structure does not imply deeply immutable elements.
- **Termination:** requires a decreasing structural argument or other termination evidence; ordinary arrow types permit divergence and exceptions.

For this review no code is executed and no test result is asserted. The runtime checker does not turn this prose into a proof. The repair must be checked and exercised on the actual library before claiming implementation correctness.
