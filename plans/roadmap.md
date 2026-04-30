# pb3-gen-sol Roadmap

This file is the durable index of remaining work for `pb3-gen-sol`. It is intended to stand on its own after short-lived notes and one-off discussions are removed.

The release milestone is intentionally last. The goal is to ship one modernized, correct, and gas-efficient version rather than rush out an earlier but slower release.

Use this roadmap together with the detailed execution plans:

- Gas and benchmarking work: [gas-optimization-plan.md](./gas-optimization-plan.md)

## Active Workstreams

### 1. Benchmark Harness and Hot-Path Gas Work — **complete**

Status: Phase 1 (`decVarint` fast path, fixed-width readers, `unchecked`
loop cleanup) and Phase 3 §8 in full (`cntTags` removed; single-pass
scratch for both inline-primitive and reference-typed repeated fields)
all shipped 2026-04-29. The current `New` runtime is **36–58% faster
than the frozen `agent-pay-contracts` runtime** on every measured path
(−50.8% aggregate). See
[benchmarks/baseline.md](./benchmarks/baseline.md) for the full table.

Primary tracking: [gas-optimization-plan.md](./gas-optimization-plan.md)

Exit criteria:

- [x] Benchmark baselines exist for representative decode paths.
- [x] `decVarint` improvements measured, not assumed.
- [x] Hot-path gas work preserves deterministic malformed-input behavior.
- [x] CI regenerates `bench/new/` and fails on drift.
- [x] §8 second half (reference-type single-pass) landed; `cntTags` removed.

### 2. Structural Runtime Optimization — **evaluated, closed**

Status: §6 (zero-copy nested submessage decoding via `decSubBuffer`)
was prototyped end-to-end on 2026-04-29 and rejected after measurement.
Aggregate AgentPay improvement was only −1.4%, with one path regressing
+3% and a stress path regressing +2.2%. The Buffer struct expansion
plus `decX → _decX` forwarder pattern impose a per-call fixed cost
that doesn't pay off on paths with few or zero nested decodes —
shallow-schema consumers would pay net regression. Stop criterion #1
("remaining ideas only produce marginal gains relative to added
complexity") triggered.

Primary tracking: [gas-optimization-plan.md § Tier B Evaluation](./gas-optimization-plan.md#tier-b-evaluation).

Closed items:

- ~~Zero-copy nested submessage decoding (§6)~~ — prototyped, rejected.
- ~~Calldata-native runtime path (§5)~~ — not pursued (Tier B as a
  whole rejected; §5 alone would have an even worse upside-to-complexity
  ratio than §6).

Dormant items (open in principle, no active work planned):

- **Partial decoders for measured hot paths (§7).** Only if the
  consumer side has paths that demonstrably need a subset of fields.
  Adds generator surface; should be driven by data, not preemptive.
- **Packed repeated tightening (§4).** Current bench shows no AgentPay
  path is bottlenecked on `decPacked`. Revisit only when a consumer
  schema introduces a hot packed-varint path.

When to reopen this workstream:

- A future consumer schema introduces materially different shapes
  (much deeper nesting than AgentPay, or a hot path bottlenecked on
  large-payload calldata copies) where the §6 / §5 economics would
  flip. Re-measure against fresh baselines before any structural
  rewrite.

Exit criteria (achieved by the closure decision):

- [x] Any structural runtime change is justified by representative benchmark improvements — §6 was not, so it was reverted.
- [x] Runtime correctness and wire compatibility remain unchanged — final shipping state is the post-§8b runtime.
- [x] Complexity stays bounded to the paths that actually need it — Tier B's per-call overhead would have spread complexity across all paths for marginal aggregate gain.

### 3. Testing Depth and Regression Coverage

Status: planned, medium priority.

Primary tracking: this roadmap and [gas-optimization-plan.md](./gas-optimization-plan.md)

Scope:

- Add at least one round-trip style regression path that verifies decode behavior end to end.
- Add mutation-oriented malformed-input coverage for nested-length issues, packed-field corruption, and related parser edges.
- Evaluate lightweight fuzzing in Go tests, Solidity tests, or both.
- Add small missing negative cases where they remain useful, including unsupported-but-enum-valid wire types in `skipValue`.
- Keep both `importpb=true` and `importpb=false` generation paths covered by integration tests.

Exit criteria:

- At least one round-trip regression test exists for a representative schema.
- Additional malformed-input coverage lands without introducing flaky CI behavior.
- Both runtime embedding modes remain protected by tests.

### 4. Generator UX and Output Quality

Status: planned, medium priority.

Primary tracking: this roadmap.

Scope:

- Decide how proto leading comments should flow into generated docblocks.
- Return structured `CodeGeneratorResponse.Error` values for supported user-facing failures instead of relying on generic plugin exits.
- If real schemas require it, scope `(soltype)` extension lookup to the current file's transitive import graph rather than the full descriptor set.
- Revisit small clarity improvements in generator internals when touching that code again, such as explaining or simplifying non-obvious package-resolution behavior.

Exit criteria:

- Generated docblocks follow an explicit, documented comment strategy.
- User-facing generator failures produce actionable diagnostics.
- Extension lookup behavior is either narrowed or clearly documented and covered.

### 5. CI and Developer Workflow Efficiency — **partially landed**

Status: planned, medium priority.

Primary tracking: this roadmap and [gas-optimization-plan.md](./gas-optimization-plan.md)

What landed:

- [x] CI regenerates both `test/solidity/src/lib` and
  `test/solidity/test/bench/new` on every PR and fails on any diff
  against the checked-in copy. The bench numbers therefore always
  reflect the current generator/runtime.
- [x] `importpb=true` and `importpb=false` generation paths both
  exercised in the integration tests.

Remaining scope:

- Cache Go modules and Foundry artifacts if the cache meaningfully improves CI time without destabilizing builds.
- Add benchmark **diff reporting** PR-over-PR (e.g., a comment with
  the deltas) once the optimization workstream stabilizes. Distinct
  from the drift check above, which is already in place.

Exit criteria:

- CI remains deterministic.
- Caching is only enabled where it is measurably useful.
- Bench diff reporting available when there is a workstream that needs it.

### 6. Release and Adoption Follow-Up

Status: planned, final milestone.

Primary tracking: this roadmap.

Scope:

- Cut a single post-modernization release only after the correctness, performance, and workflow work above are complete enough to support shipping with confidence.
- Publish release notes that describe the supported workflow and compatibility expectations for the current generator output.
- Publish migration notes for downstream users of the old `pb.sol` layout and Solidity 0.5 era output.
- If the private repo is deprecated, publish a migration note that points users to this repo and the release.
- If release automation is worth the maintenance cost, package tagged plugin binaries.

Exit criteria:

- The benchmarked gas work is complete enough that the release is not knowingly shipping the slow interim decoder path as the intended long-term version.
- Correctness and regression coverage are green across the chosen release scope.
- Release notes describe the supported workflow and compatibility expectations.
- Migration guidance exists for downstream users.
- The deprecation path is documented if the private repo is retired.
- The release is cut.

## Small Backlog Items

These are intentionally lower priority than the main workstreams above, but they should not be lost.

### 1. Narrow `(soltype)` conflict scanning to reachable imports only

Status: backlog, low priority.

Why it matters:

- The current behavior scans the full descriptor set and can fail conservatively on unrelated conflicting definitions.
- The current implementation is safe, but slightly stricter than the minimum proto scoping would require.

Exit criteria:

- Either the lookup is narrowed to reachable imports, or the current behavior is documented as an intentional conservative choice.

### 2. Add one more regression for valid-enum-member but unsupported wire types

Status: backlog, low priority.

Why it matters:

- The suite already covers the important parser edges, but one additional case would make `skipValue` behavior more explicit.

Exit criteria:

- A focused regression test exists for at least one unsupported wire type that is still within the Solidity enum range.

### 3. Clarify the same-length package-name tiebreak if that code is touched again

Status: backlog, low priority.

Why it matters:

- The current behavior is correct, but the ordering can look more meaningful than it actually is to future readers.

Exit criteria:

- The code is either simplified or carries a short explanation when that area is next modified.