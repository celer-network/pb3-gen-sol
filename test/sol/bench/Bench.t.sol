// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import {PbChain} from "../../lib/PbChain.sol";
import {PbEntity} from "../../lib/PbEntity.sol";

interface BenchVm {
    function readFileBinary(string calldata path) external returns (bytes memory);
}

/**
 * @title Bench
 * @notice Decode-gas benchmarks on the AgentPay schemas (chain.proto +
 *  entity.proto), the de-facto reference consumer for `pb3-gen-sol`. Each
 *  representative path is measured twice:
 *
 *   - `pb_decX` decodes the proto-encoded `.pb` fixture via the generated
 *     `PbEntity.decX` / `PbChain.decX` runtime.
 *   - `abi_decX` decodes the same logical struct, re-encoded via Solidity's
 *     native `abi.encode`, through `abi.decode`.
 *
 *  Same Solidity struct shape, two wire formats. The numbers expose:
 *
 *   - Per-path proto vs ABI gas cost.
 *   - Per-path proto vs ABI payload size (proto is typically much smaller
 *     for sparse messages because proto3 omits zero-valued fields and
 *     uses varint for scalars; ABI's offset-tail encoding has fixed
 *     overhead).
 *
 *  Output is `<label> <payloadBytes> <gasUsed>` per test, surfaced via
 *  Foundry's console-log precompile (no forge-std dependency). Run with
 *  `forge test --match-path 'sol/bench/Bench.t.sol' -vv`.
 */
contract BenchTest {
    address private constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    BenchVm private constant vm = BenchVm(VM_ADDRESS);

    address private constant CONSOLE = 0x000000000000000000636F6e736F6c652e6c6f67;

    function _bench(string memory label, uint256 payloadBytes, uint256 startGas, uint256 endGas) internal view {
        bytes memory payload =
            abi.encodeWithSignature("log(string,uint256,uint256)", label, payloadBytes, startGas - endGas);
        (bool ok,) = CONSOLE.staticcall(payload);
        ok; // discard: console precompile is a no-op outside Foundry
    }

    function _load(string memory name) internal returns (bytes memory) {
        return vm.readFileBinary(string.concat("fixtures/bin/", name));
    }

    // -------------------------------------------------------------------------
    // entity.AccountAmtPair: simplest representative shape (1 address + 1 uint256)
    // -------------------------------------------------------------------------

    function test_pb_decAccountAmtPair() external {
        bytes memory raw = _load("bench_account_amt_pair.pb");
        uint256 g0 = gasleft();
        PbEntity.AccountAmtPair memory m = PbEntity.decAccountAmtPair(raw);
        uint256 g1 = gasleft();
        _bench("pb_decAccountAmtPair", raw.length, g0, g1);
        require(m.amt == 1_000_000, "amt");
    }

    function test_abi_decAccountAmtPair() external {
        PbEntity.AccountAmtPair memory original = PbEntity.decAccountAmtPair(_load("bench_account_amt_pair.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.AccountAmtPair memory m = abi.decode(raw, (PbEntity.AccountAmtPair));
        uint256 g1 = gasleft();
        _bench("abi_decAccountAmtPair", raw.length, g0, g1);
        require(m.amt == 1_000_000, "amt");
    }

    // -------------------------------------------------------------------------
    // entity.TokenDistribution: nested TokenInfo + repeated AccountAmtPair
    // -------------------------------------------------------------------------

    function test_pb_decTokenDistribution() external {
        bytes memory raw = _load("bench_token_distribution.pb");
        uint256 g0 = gasleft();
        PbEntity.TokenDistribution memory m = PbEntity.decTokenDistribution(raw);
        uint256 g1 = gasleft();
        _bench("pb_decTokenDistribution", raw.length, g0, g1);
        require(m.distribution.length == 2, "distribution");
    }

    function test_abi_decTokenDistribution() external {
        PbEntity.TokenDistribution memory original = PbEntity.decTokenDistribution(_load("bench_token_distribution.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.TokenDistribution memory m = abi.decode(raw, (PbEntity.TokenDistribution));
        uint256 g1 = gasleft();
        _bench("abi_decTokenDistribution", raw.length, g0, g1);
        require(m.distribution.length == 2, "distribution");
    }

    // -------------------------------------------------------------------------
    // entity.ConditionalPay: the headline AgentPay path — 3 mixed-type
    // Conditions + nested TransferFunction + scalars.
    // -------------------------------------------------------------------------

    function test_pb_decConditionalPay() external {
        bytes memory raw = _load("bench_conditional_pay.pb");
        uint256 g0 = gasleft();
        PbEntity.ConditionalPay memory m = PbEntity.decConditionalPay(raw);
        uint256 g1 = gasleft();
        _bench("pb_decConditionalPay", raw.length, g0, g1);
        require(m.conditions.length == 3, "conditions");
    }

    function test_abi_decConditionalPay() external {
        PbEntity.ConditionalPay memory original = PbEntity.decConditionalPay(_load("bench_conditional_pay.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.ConditionalPay memory m = abi.decode(raw, (PbEntity.ConditionalPay));
        uint256 g1 = gasleft();
        _bench("abi_decConditionalPay", raw.length, g0, g1);
        require(m.conditions.length == 3, "conditions");
    }

    // -------------------------------------------------------------------------
    // entity.SimplexPaymentChannel: bytes32 + address + nested TokenTransfer +
    // nested PayIdList (with 2 bytes32 ids + nextListHash) + scalars.
    // -------------------------------------------------------------------------

    function test_pb_decSimplexPaymentChannel() external {
        bytes memory raw = _load("bench_simplex_payment_channel.pb");
        uint256 g0 = gasleft();
        PbEntity.SimplexPaymentChannel memory m = PbEntity.decSimplexPaymentChannel(raw);
        uint256 g1 = gasleft();
        _bench("pb_decSimplexPaymentChannel", raw.length, g0, g1);
        require(m.pendingPayIds.payIds.length == 2, "payIds");
    }

    function test_abi_decSimplexPaymentChannel() external {
        PbEntity.SimplexPaymentChannel memory original =
            PbEntity.decSimplexPaymentChannel(_load("bench_simplex_payment_channel.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.SimplexPaymentChannel memory m = abi.decode(raw, (PbEntity.SimplexPaymentChannel));
        uint256 g1 = gasleft();
        _bench("abi_decSimplexPaymentChannel", raw.length, g0, g1);
        require(m.pendingPayIds.payIds.length == 2, "payIds");
    }

    // -------------------------------------------------------------------------
    // entity.PaymentChannelInitializer: nested TokenDistribution + scalars.
    // -------------------------------------------------------------------------

    function test_pb_decPaymentChannelInitializer() external {
        bytes memory raw = _load("bench_payment_channel_initializer.pb");
        uint256 g0 = gasleft();
        PbEntity.PaymentChannelInitializer memory m = PbEntity.decPaymentChannelInitializer(raw);
        uint256 g1 = gasleft();
        _bench("pb_decPaymentChannelInitializer", raw.length, g0, g1);
        require(m.initDistribution.distribution.length == 2, "distribution");
    }

    function test_abi_decPaymentChannelInitializer() external {
        PbEntity.PaymentChannelInitializer memory original =
            PbEntity.decPaymentChannelInitializer(_load("bench_payment_channel_initializer.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.PaymentChannelInitializer memory m = abi.decode(raw, (PbEntity.PaymentChannelInitializer));
        uint256 g1 = gasleft();
        _bench("abi_decPaymentChannelInitializer", raw.length, g0, g1);
        require(m.initDistribution.distribution.length == 2, "distribution");
    }

    // -------------------------------------------------------------------------
    // entity.CooperativeWithdrawInfo: 1 nested AccountAmtPair + scalars.
    // -------------------------------------------------------------------------

    function test_pb_decCooperativeWithdrawInfo() external {
        bytes memory raw = _load("bench_cooperative_withdraw_info.pb");
        uint256 g0 = gasleft();
        PbEntity.CooperativeWithdrawInfo memory m = PbEntity.decCooperativeWithdrawInfo(raw);
        uint256 g1 = gasleft();
        _bench("pb_decCooperativeWithdrawInfo", raw.length, g0, g1);
        require(m.withdraw.amt == 50_000, "amt");
    }

    function test_abi_decCooperativeWithdrawInfo() external {
        PbEntity.CooperativeWithdrawInfo memory original =
            PbEntity.decCooperativeWithdrawInfo(_load("bench_cooperative_withdraw_info.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.CooperativeWithdrawInfo memory m = abi.decode(raw, (PbEntity.CooperativeWithdrawInfo));
        uint256 g1 = gasleft();
        _bench("abi_decCooperativeWithdrawInfo", raw.length, g0, g1);
        require(m.withdraw.amt == 50_000, "amt");
    }

    // -------------------------------------------------------------------------
    // entity.CooperativeSettleInfo: 2 repeated AccountAmtPair + scalars.
    // -------------------------------------------------------------------------

    function test_pb_decCooperativeSettleInfo() external {
        bytes memory raw = _load("bench_cooperative_settle_info.pb");
        uint256 g0 = gasleft();
        PbEntity.CooperativeSettleInfo memory m = PbEntity.decCooperativeSettleInfo(raw);
        uint256 g1 = gasleft();
        _bench("pb_decCooperativeSettleInfo", raw.length, g0, g1);
        require(m.settleBalance.length == 2, "settleBalance");
    }

    function test_abi_decCooperativeSettleInfo() external {
        PbEntity.CooperativeSettleInfo memory original =
            PbEntity.decCooperativeSettleInfo(_load("bench_cooperative_settle_info.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbEntity.CooperativeSettleInfo memory m = abi.decode(raw, (PbEntity.CooperativeSettleInfo));
        uint256 g1 = gasleft();
        _bench("abi_decCooperativeSettleInfo", raw.length, g0, g1);
        require(m.settleBalance.length == 2, "settleBalance");
    }

    // -------------------------------------------------------------------------
    // chain.ResolvePayByConditionsRequest: wrapper with bytes payload + 3 sigs.
    // -------------------------------------------------------------------------

    function test_pb_decResolvePayByConditionsRequest() external {
        bytes memory raw = _load("bench_resolve_pay_request.pb");
        uint256 g0 = gasleft();
        PbChain.ResolvePayByConditionsRequest memory m = PbChain.decResolvePayByConditionsRequest(raw);
        uint256 g1 = gasleft();
        _bench("pb_decResolvePayByConditionsRequest", raw.length, g0, g1);
        require(m.hashPreimages.length == 3, "hashPreimages");
    }

    function test_abi_decResolvePayByConditionsRequest() external {
        PbChain.ResolvePayByConditionsRequest memory original =
            PbChain.decResolvePayByConditionsRequest(_load("bench_resolve_pay_request.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbChain.ResolvePayByConditionsRequest memory m = abi.decode(raw, (PbChain.ResolvePayByConditionsRequest));
        uint256 g1 = gasleft();
        _bench("abi_decResolvePayByConditionsRequest", raw.length, g0, g1);
        require(m.hashPreimages.length == 3, "hashPreimages");
    }

    // -------------------------------------------------------------------------
    // chain.SignedSimplexState: wrapper with bytes simplex payload + 2 sigs.
    // -------------------------------------------------------------------------

    function test_pb_decSignedSimplexState() external {
        bytes memory raw = _load("bench_signed_simplex_state.pb");
        uint256 g0 = gasleft();
        PbChain.SignedSimplexState memory m = PbChain.decSignedSimplexState(raw);
        uint256 g1 = gasleft();
        _bench("pb_decSignedSimplexState", raw.length, g0, g1);
        require(m.sigs.length == 2, "sigs");
    }

    function test_abi_decSignedSimplexState() external {
        PbChain.SignedSimplexState memory original =
            PbChain.decSignedSimplexState(_load("bench_signed_simplex_state.pb"));
        bytes memory raw = abi.encode(original);
        uint256 g0 = gasleft();
        PbChain.SignedSimplexState memory m = abi.decode(raw, (PbChain.SignedSimplexState));
        uint256 g1 = gasleft();
        _bench("abi_decSignedSimplexState", raw.length, g0, g1);
        require(m.sigs.length == 2, "sigs");
    }
}
