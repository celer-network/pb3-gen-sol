# Decode Gas Baseline

Captured 2026-04-29 at the head of the `improve` branch, before any Phase 1
gas work has landed.

For setup and reproduction instructions, see [README.md](./README.md).

## Toolchain

- `forge` 1.5.1-v1.5.1 (`b0a9dd9c`, build profile `maxperf`)
- `solc` 0.8.30 (selected by Foundry; `solc-bin` 0.8.33 also confirmed
  produces the same numbers)
- `optimizer = true`, `optimizer_runs = 200`
- protoc `libprotoc 33.2`
- Plugin built from working tree, branch `improve`

The `Old` half is the frozen [hand-tuned AgentPay
runtime](https://github.com/celer-network/agent-pay-contracts/blob/main/src/lib/data/Pb.sol);
the `New` half is the current generator output.

## Decode-only gas

Numbers are gas spent inside the decode call only (`gasleft()` deltas
around just the `dec*` invocations). Payload construction, the bench's
own assertion calls, and console-log emission are all excluded.

| Path                                    | Bytes | Old gas | New gas | Δ gas | Δ %     |
| --------------------------------------- | -----:| -------:| -------:| -----:| -------:|
| `decConditionalPay`                     |   185 |  57,401 |  58,995 | +1,594 | +2.78%  |
| `decSimplexPaymentChannel`              |   182 |  29,399 |  29,201 |   −198 | −0.67%  |
| `decPaymentChannelInitializer`          |    72 |  26,571 |  26,611 |    +40 | +0.15%  |
| `decCooperativeWithdrawInfo`            |    69 |  11,043 |  11,226 |   +183 | +1.66%  |
| `decCooperativeSettleInfo`              |    99 |  26,142 |  26,640 |   +498 | +1.91%  |
| `decResolvePayRequest + decConditionalPay` | 290 |  73,606 |  75,286 | +1,680 | +2.28%  |
| `decSignedSimplexState + decSimplex`    |   319 |  42,542 |  42,498 |    −44 | −0.10%  |
| **Sum across all paths**                |       | 266,704 | 270,457 | +3,753 | +1.41%  |

## What this tells us

- **The `New` runtime is within ~3% of the legacy hand-tuned `Old`
  runtime on every measured path.** This is much closer than the round-1
  review's qualitative prediction. The hardening (per-byte
  bounds-checked `decVarint`, `cntTags` unknown-tag guard) costs noticeably
  less than expected in absolute terms.
- **The biggest absolute gap is `decConditionalPay`** (+1,594 gas). It has
  the most varint-heavy primitive fields (`payTimestamp`, `resolveDeadline`,
  `resolveTimeout`) plus three nested `Condition` submessages. Both
  factors compound the per-byte `decVarint` overhead. The end-to-end
  `ResolvePayRequest → ConditionalPay` path inherits the same gap.
- **Two paths are slightly faster on `New`** (`decSimplexPaymentChannel`,
  `decSignedSimplexState + decSimplex`). The deltas are tiny (≤0.7%) and
  most likely from incidental code-shape differences after the recent
  generator cleanups (no `if (false) {}` sentinel, fewer identity casts);
  not from any algorithmic improvement.
- **No path regresses by more than 3%.** The hardening tax is real but
  bounded.

## Implications for Phase 1 ordering

1. **Hardened-fast `decVarint` is still the highest-value target**, but
   the upside is modest in absolute terms — a 50% speedup on `decVarint`
   itself would shave roughly 1–2 kgas off `decConditionalPay`, not 10
   kgas. Re-prioritize: it's a "close the gap" change, not a "step
   change."
2. **Fixed-width readers for `address` / `bytes32` / `uint256-bytes` are
   likely the bigger win.** Every `decConditionalPay` payload allocates
   `bytes` for ~6 fixed-width primitive fields (3 addresses + 1 bytes32
   per Condition × 3 Conditions); each allocation+`mload`+drop is much
   more expensive than the varint hardening overhead. Worth measuring
   that path explicitly.
3. **The `cntTags` double-pass cost** is included in every measurement
   above for paths that have repeated length-delimited fields
   (`PaymentChannelInitializer`, `CooperativeSettleInfo`,
   `ConditionalPay`'s `conditions`, the `Resolve*` and `Signed*`
   wrappers). It's the dominant fixed cost on small messages and may
   merit its own follow-up benchmark.
4. **Decode for `decCooperativeWithdrawInfo` is only 11 kgas** — at this
   size, even tens of gas per varint matter proportionally. This is the
   right "small message" path to watch when verifying that the fast-path
   `decVarint` doesn't accidentally regress short-message latency.

## Next steps

- The harness is in place. Phase 1 work
  ([gas-optimization-plan.md](../gas-optimization-plan.md), §1) can start.
- Treat this baseline as a frozen reference. Each optimization PR should
  rerun the bench, paste the new numbers, and update this file (preserving
  the dated baseline as a comparison row).
