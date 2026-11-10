# Review of the stack claim

The conclusion does not follow. Datatype axioms over untyped operation names
are a trusted interface, not a check of their implementation bodies. Clients can
prove consequences of a false axiom. A `declare` would have the same limitation.

## Smallest counterexample

Start with `S = empty`, push one value `7`, then ask for the top of the result.
The asserted interface gives `push 7 S : stack number [succ 0]`, so the client
call to `top` typechecks. But the given push body returns `S`, which is still
empty. There is no legitimate newest `7` to return. Depending on the unprovided
empty/top implementation it might fail, loop, or return some bogus default. No
specific execution result is claimed here. A minimal specification test requires
`top (push 7 empty) = 7` and must raise an error on mismatch; an exception is
also a failure, not a pass. This single push suffices to refute the growth claim.

## Repair and evidence

* Fix push to construct a new cell or closure containing the item and the old
  stack. Do not mutate the old stack if persistence is required.
* Prefer a structural representation with an empty base rule and a double-line
  cons rule indexed by successor. Give **implementation bodies** signatures for
  empty, push, top, and pop. Then `push X S -> S` must be rejected as claiming a
  successor index while returning the old index. Check this deliberately wrong
  implementation, not merely a wrong client.
* Add negative clients for top-of-empty and top-after-popping-a-singleton. Add
  positive assertions for top-after-push, LIFO order with two distinct values,
  pop restoring the predecessor, and retained old versions staying unchanged.
  These are finite tests, not universal proofs.
* If opaque untyped closures must remain, minimize and explicitly document the
  trusted boundary. Validate it with model-based tests against immutable lists,
  implementation review, or a separate simulation proof. Publishing operation
  axioms alone cannot discharge those obligations.

## What the indices actually establish

With sound representation rules and checked bodies, these signatures can track
structural length and make top require a nonempty stack. They do not specify
which element top returns, that push stores its argument, or which predecessor
pop restores. An implementation could keep the length while reordering or
replacing elements. Ordering needs a content/sequence model or separate laws;
persistence needs old-version preservation; purity needs effect reasoning or
inspection of the operations and their reachable state; termination needs a
well-founded argument or other termination evidence. Shen arrow types and
length indices do not provide these automatically.

Any proof still assumes sound inference rules, correct primitive/runtime
behavior, no hostile datatype extension or redefinition, and sound declarations.
A checked structural repair reduces the trusted surface; it does not turn tests
or those assumptions into theorems. This review is prose only and reports no
invented run of the unprovided closure library.
