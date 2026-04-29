# Benchmarks

Foundry-driven gas baselines for the `pb3-gen-sol` runtime, focused on the
representative AgentPay decode paths called out in
[`gas-optimization-plan.md`](../gas-optimization-plan.md).

## What this measures

Each path is benchmarked against two runtimes in the same Foundry suite:

- **`*Old`** — frozen snapshot of the hand-tuned runtime and decoders shipped
  in [`agent-pay-contracts/src/lib/data/`](https://github.com/celer-network/agent-pay-contracts/tree/main/src/lib/data).
  Single-`mload` `decVarint`, no `cntTags` unknown-tag guard, two-allocation
  `decPacked`. Already deployed.
- **`*New`** — what `pb3-gen-sol` emits today against the same proto
  schemas. Per-byte bounds-checked `decVarint`, `cntTags` unknown-tag guard,
  in-place-shrink `decPacked`. Hardened, not yet gas-tuned.

The `Old` half exists so every optimization PR can show three deltas:
versus the previous `New` baseline (does this PR help?), versus the prior
`Old` baseline (have we caught up to the legacy runtime?), and versus
each other path (which paths benefit?).

## Layout

- [bench/proto/](../../test/solidity/test/bench/proto/) — `chain.proto` and
  `entity.proto` copied from `agent-pay-contracts`. Self-contained, so the
  bench can be regenerated without an external sibling checkout.
- [bench/old/](../../test/solidity/test/bench/old/) — frozen `Pb.sol`,
  `PbChain.sol`, `PbEntity.sol` from the AgentPay repo. Libraries are
  renamed `*Old` so they coexist with the `*New` half. **Do not regenerate.**
- [bench/new/](../../test/solidity/test/bench/new/) — runtime + decoders
  emitted by the current generator. Libraries renamed `*New`.
- [bench/util/](../../test/solidity/test/bench/util/) — `Proto.sol`,
  `Fixtures.sol` builders copied from
  `agent-pay-contracts/test/utils/`. Used by both halves of the bench.
- [bench/Decode.t.sol](../../test/solidity/test/bench/Decode.t.sol) —
  paired benchmark functions for each representative path.
- [bench/regen.sh](../../test/solidity/test/bench/regen.sh) — rebuilds the
  plugin and regenerates `bench/new/` (preserving the `*New` library names).

## Running the bench

```bash
cd test/solidity
forge test --match-path 'test/bench/Decode.t.sol' -vv \
  | grep -E '_(old|new) [0-9]'
```

Each line is `<label> <payloadBytes> <gasUsed>`. Decode-only gas (no
payload-construction or assertion overhead).

## Updating the `New` half after a runtime change

```bash
bash test/solidity/test/bench/regen.sh
cd test/solidity && forge test --match-path 'test/bench/Decode.t.sol' -vv
```

The regen script builds the plugin from the current source, runs `protoc`
against `bench/proto/`, and applies the `Pb` → `PbNew` rename. Idempotent.

The `bench/old/` snapshot is intentionally not touched by the script;
treat that directory as immutable.

## Capturing a new baseline

After landing an optimization PR, update [baseline.md](./baseline.md) with
the latest numbers. Keep a short prose note about what changed so future
readers can connect the deltas to the runtime change.
