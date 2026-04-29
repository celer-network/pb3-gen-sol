# pb3-gen-sol Roadmap

This file is the durable index of remaining work for `pb3-gen-sol`. It is intended to stand on its own after short-lived notes and one-off discussions are removed.

The release milestone is intentionally last. The goal is to ship one modernized, correct, and gas-efficient version rather than rush out an earlier but slower release.

Use this roadmap together with the detailed execution plans:

- Gas and benchmarking work: [gas-optimization-plan.md](./gas-optimization-plan.md)

## Active Workstreams

### 1. Benchmark Harness and Hot-Path Gas Work

Status: planned, high priority.

Primary tracking: [gas-optimization-plan.md](./gas-optimization-plan.md)

Scope:

- Add the Foundry benchmark harness and store baseline measurements.
- Rework `decVarint` only behind measurement, while preserving current malformed-input behavior.
- Add fixed-width readers for bytes-backed primitives such as `address`, `bytes32`, and bytes-backed `uint256`.
- Apply small runtime loop cleanups only when benchmarks show a real win.

Exit criteria:

- Benchmark baselines exist for representative decode paths.
- `decVarint` improvements are measured, not assumed.
- Hot-path gas work preserves deterministic malformed-input behavior.

### 2. Structural Runtime Optimization

Status: planned, medium priority.

Primary tracking: [gas-optimization-plan.md](./gas-optimization-plan.md)

Scope:

- Evaluate a calldata-native decode path.
- Evaluate zero-copy or lower-copy nested submessage decoding.
- Add partial decoders only for measured hot paths that justify the extra generated surface.
- Revisit repeated-field strategy, including `cntTags` double-pass cost and any further packed-field tightening, only after the higher-confidence wins land.

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

### 5. CI and Developer Workflow Efficiency

Status: planned, medium priority.

Primary tracking: this roadmap and [gas-optimization-plan.md](./gas-optimization-plan.md)

Scope:

- Cache Go modules and Foundry artifacts if the cache meaningfully improves CI time without destabilizing builds.
- Add benchmark diff reporting once the benchmark harness is stable enough to trust in CI.
- Keep regeneration drift checks and integration coverage aligned with the current generation modes and checked-in artifacts.

Exit criteria:

- CI remains deterministic.
- Caching is only enabled where it is measurably useful.
- Performance reporting is available once the benchmark harness is ready.

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