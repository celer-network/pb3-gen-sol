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

### Latest — 2026-04-29 — Phase 1 §3 (`unchecked` loop counters)

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

The `New` runtime is now 30–37% faster than the legacy hand-tuned `Old`
runtime on every measured path. 32/32 tests green; every malformed-input
revert preserved.

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

- **Phase 1 is essentially done.** The per-byte and per-arithmetic
  overhead in the runtime is now minimal. Further wins require
  structural changes (Phase 2/3) — the easy "constant-factor scrubbing"
  surface is mostly gone.
- **`cntTags` double-pass** (Phase 3 §8) is now the single largest
  remaining cost on most paths. It scans the full payload twice. With
  varint and fixed-width readers already optimized, the dominant cost
  inside that pre-pass is just the `decKey` + `skipValue` call shape
  itself. Eliminating the pre-pass entirely (over-allocate + shrink for
  repeated fields, like `decPacked` already does) would be the next
  big lever.
- **Calldata-native decoding** (Phase 2 §5) becomes more interesting
  the smaller decode itself becomes. The per-call memory-copy cost on
  external entry is now a larger relative share.
- **Stop criteria check.** The plan's stop criterion #1 ("the remaining
  ideas only produce marginal gains relative to added complexity") is
  approaching for Phase 1. Whether to attempt Phase 2 should be a
  deliberate measurement-driven decision, not automatic.

## Reproduction

```bash
cd test/solidity
forge test --match-path 'test/bench/Decode.t.sol' -vv \
  | grep -E '_(old|new) [0-9]'
```

Numbers are deterministic across runs and across solc 0.8.30 / 0.8.33.

After landing the next Phase 1 PR, replace the "Latest" table above
with the new numbers and move the previous "Latest" down as a dated
historical row.
