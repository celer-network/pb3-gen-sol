# Decode Benchmark

`Bench.t.sol` measures decode gas on the AgentPay schemas
([`test/proto/bench/chain.proto`](../../proto/bench/chain.proto) +
[`test/proto/bench/entity.proto`](../../proto/bench/entity.proto)) —
copied verbatim from
[`agent-pay-contracts`](https://github.com/celer-network/agent-pay-contracts),
the de-facto reference consumer for `pb3-gen-sol`. For each
representative path two tests run:

- `pb_decX` decodes the proto-encoded `.pb` fixture via the generated
  runtime.
- `abi_decX` re-encodes the *same* Solidity struct via `abi.encode` and
  decodes it back through `abi.decode`, as a native baseline.

Both produce the same logical struct from different wire formats. The
comparison exposes protobuf-vs-ABI as a calldata/gas trade-off, not a
performance-vs-performance comparison.

## How to run

From the Foundry root (`test/`):

```bash
forge test --match-path 'sol/bench/Bench.t.sol' -vv \
  | grep -E '^  (pb|abi)_dec'
```

Each line is `<label> <payloadBytes> <gasUsed>`. Gas is decode-only
(`gasleft()` deltas around just the `dec*` call); the fixture load and
assertion calls sit outside the metering window.

## Latest numbers

Captured 2026-04-30 with `forge` 1.5.1 / `solc` 0.8.30, optimizer on at
200 runs. Refresh by re-running the bench when runtime/generator
changes meaningfully.

| Path                                     | Pb bytes | Pb gas  | ABI bytes | ABI gas | Ratio  |
| ---------------------------------------- | -------: | ------: | --------: | ------: | -----: |
| `entity.AccountAmtPair`                  |       27 |   2,285 |        64 |     428 |  5.34× |
| `entity.TokenDistribution`               |       84 |  11,196 |       288 |   1,941 |  5.77× |
| `entity.ConditionalPay`                  |      185 |  27,009 |     1,376 |   6,550 |  4.12× |
| `entity.SimplexPaymentChannel`           |      182 |  15,343 |       512 |   2,272 |  6.75× |
| `entity.PaymentChannelInitializer`       |       72 |  13,848 |       416 |   2,231 |  6.21× |
| `entity.CooperativeWithdrawInfo`         |       69 |   6,993 |       192 |     646 | 10.83× |
| `entity.CooperativeSettleInfo`           |       99 |  11,039 |       320 |   1,614 |  6.84× |
| `chain.ResolvePayByConditionsRequest`    |      136 |   6,056 |       512 |   2,699 |  2.24× |
| `chain.SignedSimplexState`               |      175 |   5,190 |       544 |   2,235 |  2.32× |

`Ratio = Pb gas / ABI gas`. Higher means the protobuf decode is more
expensive relative to ABI on that shape.

## How to read the trade-off

Two things shift across paths:

**Wire size.** Protobuf is 0.13–0.42× the size of the equivalent ABI
encoding on these messages. Proto3 omits zero-valued fields and packs
scalars as varints; ABI uses fixed 32-byte slot widths in the head and
offset-tail layout for dynamic fields. Smaller wire size means cheaper
calldata at submission time.

**Decode gas.** ABI decode costs 2.2–10.8× less gas than the protobuf
decoder on these shapes. ABI's wire format is essentially a
pre-laid-out memory layout — decoding is mostly pointer arithmetic
over fixed-width slots. Protobuf's wire format is length-prefix-
delimited with per-field varint tags, so the decoder pays per-field
metadata overhead (tag varint + dispatch + length-prefix varint +
per-type validation) regardless of how much actual data each field
carries.

The two effects move in opposite directions, and neither dominates
universally. The 2.2–10.8× spread in the ratio reflects each message's
mix of bulk-bytes content (where Pb amortizes per-field cost across
copies) versus many small fixed-width fields (where Pb pays per-field
overhead with no amortization, and ABI just reads inline head slots).

## What this means in practice

### Gas and calldata

The total cost per call is `decode_gas + payload_bytes × calldata_gas_per_byte`.

**L1 mainnet** (16 gas per non-zero calldata byte): the two encodings
come out roughly comparable on most paths — ABI's 2–11× decode-gas
advantage is largely offset by its 2.4–7.6× larger wire size. For
mostly-fixed-width-field messages like `CooperativeWithdrawInfo`, ABI
wins by a few thousand gas per call (~8.1k vs ~3.7k). For
bulk-bytes-wrapped messages like `SignedSimplexState`, protobuf wins
(~8.0k vs ~10.9k) because its wire-size advantage outweighs the
decode-gas penalty. For mixed-shape messages like `ConditionalPay`,
the totals come within a few percent of each other (~30k either way).

**L2 rollups** have a much higher effective per-byte cost because
calldata is posted to L1 for data availability. The 0.13–0.42×
wire-size advantage from protobuf compounds; the calldata savings
typically exceed the decode-gas penalty, so protobuf wins on
virtually every shape.

### Beyond gas: protobuf's value isn't (only) about decode cost

Gas is one input to the decision, not the only one. Protobuf's value
over hand-rolled Solidity ABI structs is primarily about being a
**portable schema** with native answers to problems that ABI doesn't
solve:

- **Language-neutral.** A `.proto` schema generates encoders/decoders
  for Go, Rust, TypeScript, Java, Python, etc. via the upstream
  `protoc` ecosystem; `pb3-gen-sol` adds Solidity to that set. ABI
  struct decoders only exist in Solidity tooling — non-Solidity
  producers must reimplement Solidity's layout rules by hand.
- **Platform-neutral wire format.** A `.pb` payload is byte-identical
  regardless of which language produced it; the same signed bytes can
  be decoded by EVM contracts (via `pb3-gen-sol`), non-EVM chains
  (CosmWasm, Move, Solana programs with protobuf libraries), or any
  off-chain verifier. ABI is EVM-specific.
- **Schema evolution.** Adding new fields with new tags is safe —
  older protobuf decoders skip unknown tags per the spec. ABI has no
  native skip-unknown: adding a field to a Solidity struct shifts
  every offset and breaks every existing decoder. The runtime
  explicitly preserves proto3 skip semantics, including for wire types
  it doesn't generate code for (`Fixed32` / `Fixed64`).
- **Field absence vs zero.** Proto3 elides zero-valued fields
  entirely; ABI requires every field present in the encoding,
  inflating the wire size of sparse messages.

If the off-chain side is already a multi-language system using
protobuf end-to-end ([AgentPay's case](https://github.com/celer-network/agent-pay-contracts)),
the on-chain decoder's gas cost is the price of keeping a single
consistent message definition across the stack — usually worth more
in maintenance and correctness than it costs in gas.

If you control both sides and only target Solidity, ABI is the simpler
default.

The bench numbers above are inputs to the decision; they aren't the
decision.

## Why the runtime is shaped the way it is

The current decoder is the result of an iterative gas-optimization
pass: hardened-fast `decVarint`, fixed-width readers
(`decAddress` / `decBytes32` / `decUint256`), `unchecked` loop
counters, and a single-pass scratch-array strategy for repeated fields
(typed scratch for inline primitives like `bytes32` / `address` /
`uint256`; typeless `uint256[]` scratch + assembly alias for reference
types like `bytes` / `string` / embedded structs). `cntTags` was
removed entirely.

Anyone digging into the decoder's design choices: the inline comments
in `generator/Pb.runtime.sol` document the *why* of each unsafe-looking
mload/byte-extraction trick, and `generator/generator.go` carries the
matching code-shape rationale on `repeatedField`, `getSolDecodeStr`,
and friends.
