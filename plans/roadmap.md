# pb3-gen-sol Roadmap

This file is the durable index of remaining work for `pb3-gen-sol`. It is intended to stand on its own after short-lived notes and one-off discussions are removed.

The release milestone is intentionally last. The goal is to ship one modernized, correct, and gas-efficient version rather than rush out an earlier but slower release.

Use this roadmap together with the detailed execution plans:

- Gas and benchmarking work: [gas-optimization-plan.md](./gas-optimization-plan.md)

## Active Workstreams

### 1. Benchmark Harness and Hot-Path Gas Work — **substantially complete**

Status: Phase 1 (`decVarint` fast path, fixed-width readers, `unchecked`
loop cleanup) and the inline-primitive half of Phase 3 §8 (`cntTags`
single-pass for `bytes32` / `address` / `uint256` repeated fields) all
shipped 2026-04-29. The current `New` runtime is **30–48% faster than
the frozen `agent-pay-contracts` runtime** on every measured path
(−34.7% aggregate). See
[benchmarks/baseline.md](./benchmarks/baseline.md) for the full table.

Primary tracking: [gas-optimization-plan.md](./gas-optimization-plan.md)

Remaining within this workstream:

- Single-pass for **reference-typed** repeated fields (`bytes` / `string`
  / embedded struct) via a typeless `uint256[]` scratch + assembly
  alias. Closes out §8. Estimated upside ~500–1,500 gas per affected
  path. Recommended next step before any structural redesign — see
  Tier A in the gas plan.

Exit criteria:

- [x] Benchmark baselines exist for representative decode paths.
- [x] `decVarint` improvements measured, not assumed.
- [x] Hot-path gas work preserves deterministic malformed-input behavior.
- [x] CI regenerates `bench/new/` and fails on drift.
- [ ] §8 second half (reference-type single-pass) lands or is explicitly closed as not worth the complexity.

### 2. Structural Runtime Optimization — **open**

Status: planned, medium priority. The two largest remaining levers
share a common precondition: a Buffer redesign that supports both
calldata-backed reads and offset-based slicing into a parent buffer.

Primary tracking: [gas-optimization-plan.md](./gas-optimization-plan.md)

Recommended order (revised post-§8):

1. **Zero-copy nested submessage decoding (§6).** Biggest remaining
   upside on AgentPay because every entrypoint has nested decodes.
   Replace `decX(buf.decBytes())` with an offset-based slice — no
   allocation per nested message. Should drive the Buffer redesign.
2. **Calldata-native runtime path (§5).** Eliminates the
   calldata→memory copy on external entrypoints. Builds on the same
   Buffer surface introduced for §6.
3. **Partial decoders for measured hot paths (§7).** Only if the
   consumer side has paths that demonstrably need a subset of fields.
   Adds generator surface; should be driven by data, not preemptive.

Earlier drafts ordered §5 / §6 ahead of §8 because §8 sat under "Phase 3
Repeated-Field Strategy". In practice §8's inline-primitive half was
contained Phase 1 work and landed cleanly; the reference-type half is
also Tier-A contained work and should ship before any of the structural
items above.

Deferred / dormant:

- **Packed repeated tightening (§4).** Current bench shows no AgentPay
  path is bottlenecked on `decPacked`. Revisit only when a consumer
  schema introduces a hot packed-varint path.

Exit criteria:

- Any structural runtime change is justified by representative benchmark improvements.
- Runtime correctness and wire compatibility remain unchanged.
- Complexity stays bounded to the paths that actually need it.

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