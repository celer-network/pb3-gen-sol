# Benchmarks

Foundry-driven gas baselines for the `pb3-gen-sol` runtime, focused on the
representative AgentPay decode paths called out in
[`gas-optimization-plan.md`](../gas-optimization-plan.md).

## What this measures

The bench has two flavors of test contract under
[test/solidity/test/bench/](../../test/solidity/test/bench/):

- **`Decode.t.sol`** — paired old-vs-new measurements on the AgentPay
  shapes from the gas plan. Each path is decoded against two runtimes in
  the same Foundry suite:

  - **`*Old`** — frozen snapshot of the hand-tuned runtime and decoders
    shipped in
    [`agent-pay-contracts/src/lib/data/`](https://github.com/celer-network/agent-pay-contracts/tree/main/src/lib/data).
    Single-`mload` `decVarint`, no `cntTags` unknown-tag guard,
    two-allocation `decPacked`. Already deployed.
  - **`*New`** — what `pb3-gen-sol` emits today against the same proto
    schemas. Hardened-fast `decVarint`, fixed-width readers, `unchecked`
    loop counters, single-pass scratch arrays for inline-primitive
    repeated fields. As of the latest baseline 30–48% faster than `Old`
    on every measured path.

  The `Old` half is intentionally frozen so each delta has a stable
  comparison point.

- **`Stress.t.sol`** — single-runtime measurements on shapes the
  AgentPay fixtures don't exercise:

  - `decMsg2`: four scratch-eligible repeated fields
    (`address` / `address payable` / `uint256`-bytes / `bytes32`),
    stresses the multi-scratch over-allocation case.
  - `decMsg3`: nested `repeated Msg1` + `repeated Msg2`, stresses the
    interaction between `cntTags` (used for reference-element repeated
    fields) and scratch arrays (used inside each nested Msg2).

  No paired `Old` runtime here — `test.proto` has no legacy snapshot.
  Numbers are kept as absolute baselines so a future regression on
  multi-scratch decoders surfaces in CI.

## Layout

- [bench/proto/](../../test/solidity/test/bench/proto/) — `chain.proto` and
  `entity.proto` copied from `agent-pay-contracts`. Self-contained, so the
  bench can be regenerated without an external sibling checkout.
- [bench/old/](../../test/solidity/test/bench/old/) — frozen `Pb.sol`,
  `PbChain.sol`, `PbEntity.sol` from the AgentPay repo. Libraries are
  renamed `*Old` so they coexist with the `*New` half. **Do not regenerate.**
- [bench/new/](../../test/solidity/test/bench/new/) — runtime + decoders
  emitted by the current generator. Libraries renamed `*New`. CI
  regenerates this directory and fails the build if the result drifts
  from the checked-in copy, so the bench numbers always reflect the
  current generator/runtime.
- [bench/util/](../../test/solidity/test/bench/util/) — `Proto.sol`,
  `Fixtures.sol` builders copied from
  `agent-pay-contracts/test/utils/`. Used by both halves of the bench.
- [bench/Decode.t.sol](../../test/solidity/test/bench/Decode.t.sol) —
  paired old-vs-new benchmark functions for each AgentPay path.
- [bench/Stress.t.sol](../../test/solidity/test/bench/Stress.t.sol) —
  multi-scratch-field and nested-message stress decoders, single
  runtime, fixture-loaded from `test/fixtures/bin/`.
- [bench/regen.sh](../../test/solidity/test/bench/regen.sh) — rebuilds
  the plugin and regenerates `bench/new/` (preserving the `*New` library
  names). Idempotent; CI runs this before the drift check.

## Running the bench

```bash
cd test/solidity
forge test --match-path 'test/bench/*.t.sol' -vv \
  | grep -E '_(old|new|multiScratch|nested) [0-9]'
```

Each line is `<label> <payloadBytes> <gasUsed>`. Decode-only gas (no
payload-construction or assertion overhead).

## Updating the `New` half after a runtime change

```bash
bash test/solidity/test/bench/regen.sh
cd test/solidity && forge test --match-path 'test/bench/*.t.sol' -vv
```

The regen script builds the plugin from the current source, runs `protoc`
against `bench/proto/`, and applies the `Pb` → `PbNew` rename. Idempotent.

The `bench/old/` snapshot is intentionally not touched by the script;
treat that directory as immutable. CI runs `regen.sh` and fails on any
drift in `bench/new/`, so a runtime change that forgets to refresh the
benchmark fixtures will be caught before merge.

## Capturing a new baseline

After landing a runtime or generator change that moves the numbers,
update [baseline.md](./baseline.md): replace the "Latest" table with
fresh measurements and demote the previous "Latest" to a dated
historical row. Keep a short prose note about what changed so future
readers can connect the deltas to the underlying change.
