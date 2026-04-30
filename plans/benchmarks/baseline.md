# Decode Gas Baseline

For setup and reproduction instructions, see [README.md](./README.md).

## Toolchain

- `forge` 1.5.1-v1.5.1 (`b0a9dd9c`, build profile `maxperf`)
- `solc` 0.8.30 (selected by Foundry; `solc-bin` 0.8.33 also confirmed
  produces the same numbers)
- `optimizer = true`, `optimizer_runs = 200`
- protoc `libprotoc 33.2`

The `Old` half is the frozen [hand-tuned AgentPay
runtime](https://github.com/celer-network/agent-pay-contracts/blob/main/src/lib/data/Pb.sol);
the `New` half is the current generator output.

## Decode-only gas

Numbers are gas spent inside the decode call only (`gasleft()` deltas
around just the `dec*` invocations). Payload construction, the bench's
own assertion calls, and console-log emission are all excluded.

### Latest — 2026-04-29 — Phase 3 §8 (single-pass for inline-primitive repeated fields)

| Path                                       | Bytes |    Old |    New |       Δ vs Old |
| ------------------------------------------ | ----: | -----: | -----: | -------------: |
| `decConditionalPay`                        |   185 | 57,401 | 39,989 | −17,412 (−30.3%) |
| `decSimplexPaymentChannel`                 |   182 | 29,399 | 15,404 | −13,995 (−47.6%) |
| `decPaymentChannelInitializer`             |    72 | 26,571 | 17,905 |  −8,666 (−32.6%) |
| `decCooperativeWithdrawInfo`               |    69 | 11,043 |  6,978 |  −4,065 (−36.8%) |
| `decCooperativeSettleInfo`                 |    99 | 26,142 | 17,283 |  −8,859 (−33.9%) |
| `decResolvePayRequest + decConditionalPay` |   290 | 73,606 | 51,750 | −21,856 (−29.7%) |
| `decSignedSimplexState + decSimplex`       |   319 | 42,542 | 24,818 | −17,724 (−41.7%) |
| **Sum across all paths**                   |       | **266,704** | **174,127** | **−92,577 (−34.7%)** |

The `New` runtime is now 30–48% faster than the legacy hand-tuned `Old`
runtime. 32/32 tests green; every malformed-input revert preserved.

### 2026-04-29 — after Phase 1 §3 (`unchecked` loop counters), pre-§8

| Path                                       | Bytes |    Old |    New |       Δ vs Old |
| ------------------------------------------ | ----: | -----: | -----: | -------------: |
| `decConditionalPay`                        |   185 | 57,401 | 39,989 | −17,412 (−30.3%) |
| `decSimplexPaymentChannel`                 |   182 | 29,399 | 19,371 | −10,028 (−34.1%) |
| `decPaymentChannelInitializer`             |    72 | 26,571 | 17,905 |  −8,666 (−32.6%) |
| `decCooperativeWithdrawInfo`               |    69 | 11,043 |  6,978 |  −4,065 (−36.8%) |
| `decCooperativeSettleInfo`                 |    99 | 26,142 | 17,283 |  −8,859 (−33.9%) |
| `decResolvePayRequest + decConditionalPay` |   290 | 73,606 | 51,750 | −21,856 (−29.7%) |
| `decSignedSimplexState + decSimplex`       |   319 | 42,542 | 28,786 | −13,756 (−32.3%) |
| **Sum across all paths**                   |       | **266,704** | **182,062** | **−84,642 (−31.7%)** |

### 2026-04-29 — after Phase 1 §2 (fixed-width readers), pre-§3

| Path                                       | Bytes |    Old |    New |       Δ vs Old |
| ------------------------------------------ | ----: | -----: | -----: | -------------: |
| `decConditionalPay`                        |   185 | 57,401 | 47,447 | −9,954 (−17.3%) |
| `decSimplexPaymentChannel`                 |   182 | 29,399 | 22,720 | −6,679 (−22.7%) |
| `decPaymentChannelInitializer`             |    72 | 26,571 | 21,101 | −5,470 (−20.6%) |
| `decCooperativeWithdrawInfo`               |    69 | 11,043 |  8,237 | −2,806 (−25.4%) |
| `decCooperativeSettleInfo`                 |    99 | 26,142 | 20,539 | −5,603 (−21.4%) |
| `decResolvePayRequest + decConditionalPay` |   290 | 73,606 | 61,712 | −11,894 (−16.2%) |
| `decSignedSimplexState + decSimplex`       |   319 | 42,542 | 34,427 | −8,115 (−19.1%) |
| **Sum across all paths**                   |       | **266,704** | **216,183** | **−50,521 (−18.9%)** |

### 2026-04-29 — after Phase 1 §1 (`decVarint` fast path), pre-§2

| Path                                       | Bytes |    Old |    New |       Δ vs Old |
| ------------------------------------------ | ----: | -----: | -----: | -------------: |
| `decConditionalPay`                        |   185 | 57,401 | 50,247 | −7,154 (−12.5%) |
| `decSimplexPaymentChannel`                 |   182 | 29,399 | 25,517 | −3,882 (−13.2%) |
| `decPaymentChannelInitializer`             |    72 | 26,571 | 23,007 | −3,564 (−13.4%) |
| `decCooperativeWithdrawInfo`               |    69 | 11,043 |  9,504 | −1,539 (−13.9%) |
| `decCooperativeSettleInfo`                 |    99 | 26,142 | 22,760 | −3,382 (−12.9%) |
| `decResolvePayRequest + decConditionalPay` |   290 | 73,606 | 64,518 | −9,088 (−12.3%) |
| `decSignedSimplexState + decSimplex`       |   319 | 42,542 | 37,230 | −5,312 (−12.5%) |
| **Sum across all paths**                   |       | **266,704** | **232,783** | **−33,921 (−12.7%)** |

### Initial — 2026-04-28 — pre-Phase-1 baseline

For reference, the numbers captured before any Phase 1 gas work:

| Path                                       | Bytes |    Old |    New |       Δ vs Old |
| ------------------------------------------ | ----: | -----: | -----: | -------------: |
| `decConditionalPay`                        |   185 | 57,401 | 58,995 | +1,594 (+2.78%) |
| `decSimplexPaymentChannel`                 |   182 | 29,399 | 29,201 |   −198 (−0.67%) |
| `decPaymentChannelInitializer`             |    72 | 26,571 | 26,611 |    +40 (+0.15%) |
| `decCooperativeWithdrawInfo`               |    69 | 11,043 | 11,226 |   +183 (+1.66%) |
| `decCooperativeSettleInfo`                 |    99 | 26,142 | 26,640 |   +498 (+1.91%) |
| `decResolvePayRequest + decConditionalPay` |   290 | 73,606 | 75,286 | +1,680 (+2.28%) |
| `decSignedSimplexState + decSimplex`       |   319 | 42,542 | 42,498 |    −44 (−0.10%) |
| **Sum across all paths**                   |       | **266,704** | **270,457** | **+3,753 (+1.41%)** |

## What this tells us

### Phase 3 §8 result (single-pass for inline-primitive repeated fields)

`cntTags` was scanning every length-delimited payload twice. §8 lets the
generator skip the pre-pass for repeated fields whose element type is an
**inline Solidity primitive** (`bytes32` / `address` / `uint256`); those
fields now over-allocate to a per-element-type upper bound
(`raw.length / 34` for `bytes32`, `/22` for `address`, `/2` for
`uint256`), fill via a per-field counter local, and `mstore`-shrink the
length at the end before assigning to the struct.

Repeated **reference** element types (`bytes`, `string`, embedded struct)
still use `cntTags`, because `new Foo[](N)` for a reference element type
zero-initializes each over-allocated slot to a fresh sub-allocation —
~8 words per `Condition`, ~2 words per empty `bytes`. An earlier
implementation that over-allocated for those types blew up
`decConditionalPay` by +9k gas; the mixed strategy avoids that
regression while keeping the win where it lands cleanly.

Result: −4.4% aggregate vs §3. The wins concentrate on paths that
decode `PayIdList` (`repeated bytes32 payIds`):

- `decSimplexPaymentChannel`: −3,967 gas (−20.5% vs §3).
- `decSignedSimplexState + decSimplex`: −3,968 gas (−13.8% vs §3).

Other paths sit at the §3 numbers — their repeated fields are reference
types and still pay the `cntTags` pre-pass cost.

### Phase 1 §3 result (`unchecked` loop counters)

Wrapped the per-iteration arithmetic in five hot loops in `unchecked`
blocks where bounds are statically enforced:

- `decVarint`'s 0..9 byte loop (counter `i`, payload index `idx`).
- `cntTags`' `cnts[tag] += 1` count update.
- `decPacked`'s element index `i`.
- `decBytes`' 32-byte stride memory copy `i += 32`.
- `uint8s`/`uint32s`/`uint64s`/`bools` array conversion counters.
- Generator-emitted `cnts[N]++` per repeated-field occurrence in every
  generated decoder.

Result: another 14.7% to 16.4% per path on top of §2. `cntTags` is the
biggest contributor — every tag in the payload pays the increment, and
the pre-pass scans the entire payload before the decode pass also pays
it. Per-decoder `cnts[N]++` adds up across nested messages.

The size of this win (similar magnitude to §1+§2 combined) was a
surprise. Two compounding effects:

1. Solidity 0.8's per-arithmetic overflow check costs ~30–40 gas, and
   these loops collectively run hundreds of iterations per `decConditionalPay`
   decode (10 varints × `decKey` × ~25 fields touched between the cntTags
   prepass and the actual decode).
2. The optimizer cannot prove the bounds itself, so the checks are
   genuinely emitted at every site. Removing them by hand maps to a real
   bytecode reduction.

### Phase 1 §2 result (fixed-width readers)

Added `decAddress` / `decBytes32` / `decUint256` runtime helpers that
read length-delimited fixed-width payloads directly from the buffer
into the target Solidity type, bypassing the intermediate `bytes`
allocation that the previous `Pb._x(buf.decBytes())` pattern paid.
Removed the now-unused `_address`, `_addressPayable`, `_uint256`,
`_bytes32` helpers. Updated the generator to emit `buf.decAddress()`
etc. directly for the bytes-backed soltype overrides.

Result: another 2,800 to 2,800 gas saved per path (5.6% to 13.3% on
top of §1), with the largest absolute wins on paths that touch many
fixed-width fields through nested submessages (`SimplexPaymentChannel`
has 7 such fields including the embedded `PayIdList`).

The pre-§2 hypothesis was "low single-digit percent" — the real
delivery is high-single-digit to low-double-digit, because each saved
allocation also saves the per-element copy in `decBytes`'s 32-byte
stride loop, plus the helper-function indirection.

### Phase 1 §1 result (`decVarint` fast path)

The `decVarint` rewrite swapped the per-iteration `bb[buf.idx]`
Solidity index access for an inline `byte(0, mload(...))` after an
explicit `idx < len` check, hoisted `buf.idx` and `bb.length` to
locals, and made the increment `unchecked`. Saved 12–14% per path vs
`Old`, materially exceeding the "close the gap" estimate. See the
2026-04-29 pre-§2 row above for that intermediate state.

The Solidity-level `bytes` index access turns out to be ~30–40 gas
per byte (implicit bounds check + `bytes1`-to-`uint8` conversion +
pointer arithmetic). With ~100 varint bytes consumed per
`decConditionalPay` decode, the per-byte savings closely match the
observed total.

### Implications for what's next

- **A single-pass strategy that also works for reference element types**
  is the natural follow-up to §8. Allocating `uint256[]` (or raw memory)
  as a typeless scratch and writing pointers in via assembly, then
  aliasing the scratch to the target array type at the end, would
  eliminate `cntTags` for `repeated bytes` / `repeated string` /
  `repeated <message>` too. Estimated upside on the agent-pay paths
  that don't already benefit: ~500–1,500 gas each, similar to the
  PayIdList-driven wins seen here. Worth doing if it lands cleanly,
  but it's more assembly per dispatch line.
- **Calldata-native decoding** (Phase 2 §5) becomes more interesting
  the smaller decode itself becomes. The per-call memory-copy cost on
  external entry is now a larger relative share of total cost.
- **Stop criteria check.** The plan's stop criterion #1 ("the remaining
  ideas only produce marginal gains relative to added complexity") is
  approaching. Each subsequent change should be a deliberate
  measurement-driven decision, not automatic.

## Stress baseline (Stress.t.sol)

Single-runtime measurements on shapes that the AgentPay paths above do
not exercise. No `Old` paired runtime — `test.proto` has no legacy
snapshot. These numbers are tracked as absolute baselines so a future
regression on multi-scratch decoders surfaces in CI.

| Path                  | Bytes |     New |
| --------------------- | ----: | ------: |
| `decMsg2_multiScratch` | 351 |  30,847 |
| `decMsg3_nested`       | 1,210 | 218,107 |

`decMsg2_multiScratch` exercises the four-scratch case directly:
`addrs` / `addrPayables` (`address`, min wire 22 → 15 slot bound),
`amts` (`uint256`-bytes, min wire 2 → 175 slot bound),
`hashes` (`bytes32`, min wire 34 → 10 slot bound). Actual occurrence
count is 3 per field, so `amts` is the worst case (172 unused slots);
the path still decodes in 88 gas/byte.

`decMsg3_nested` decodes 1,210 bytes of nested `repeated Msg1` and
`repeated Msg2`. The outer repeated fields are reference types and use
`cntTags`; each inner `Msg2` decode pays its own scratch-array cost
sized to the *inner* payload, not the outer 1,210 bytes — important
sanity check that the upper bound is taken from the right scope.

## Reproduction

```bash
cd test/solidity
forge test --match-path 'test/bench/*.t.sol' -vv \
  | grep -E '_(old|new|multiScratch|nested) [0-9]'
```

Numbers are deterministic across runs and across solc 0.8.30 / 0.8.33.

When the runtime or generator changes meaningfully, replace the
"Latest" tables above with fresh measurements and demote the previous
"Latest" to a dated historical row. CI regenerates `bench/new/` and
fails on drift, so a runtime change that forgets to refresh the
benchmark fixtures will be caught before merge.
