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

### Latest — 2026-04-29 — Phase 1 §1 (`decVarint` fast path)

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

The `New` runtime is now 12–14% faster than the legacy hand-tuned
`Old` runtime on every measured path. Every malformed-input test still
fails deterministically; total suite is 32/32 green.

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

### Phase 1 §1 result

The `decVarint` rewrite swapped the per-iteration `bb[buf.idx]`
Solidity index access for an inline `byte(0, mload(...))` after an
explicit `idx < len` check, hoisted `buf.idx` and `bb.length` to
locals, and made the increment `unchecked`.

The actual win materially exceeded the pre-implementation estimate:

- Predicted: ~1–2 kgas saved on `decConditionalPay`, "close the gap"
  with `Old`.
- Actual: 7,154 gas saved on `decConditionalPay`, **net positive vs `Old`**.

The Solidity-level `bytes` index access turns out to be much more
expensive per byte than a hand-rolled `mload` + `byte(0, ...)` — the
implicit bounds check, the `bytes1`-to-`uint8` conversion, and the
per-access pointer arithmetic add up to roughly 30–40 gas per byte
that the inline assembly path eliminates. With ~100 varint bytes
consumed in a `decConditionalPay` decode, the difference is exactly
where the savings come from.

### Implications for the rest of Phase 1

- **Fixed-width readers** (Phase 1 §2) are still worth doing, but the
  per-allocation cost is small relative to the varint cost we just
  removed. Estimate: low single-digit-percent improvement on
  fixed-width-heavy paths, not double-digit.
- **`cntTags` double-pass cost** is now relatively a larger share of
  total decode gas because per-varint cost dropped. Phase 3 §8 may be
  worth re-prioritizing once Phase 1 §2 lands.
- **Runtime loop cleanup** (Phase 1 §3) has even less headroom now;
  evaluate against the new baseline before scoping it.

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
