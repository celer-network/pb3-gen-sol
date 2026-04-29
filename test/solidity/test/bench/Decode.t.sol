// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import {BenchBase} from "./BenchBase.sol";
import {Fixtures} from "./util/Fixtures.sol";

import {PbChainOld} from "./old/PbChain.sol";
import {PbEntityOld} from "./old/PbEntity.sol";

import {PbChainNew} from "./new/PbChain.sol";
import {PbEntityNew} from "./new/PbEntity.sol";

/**
 * @title Decode benchmark suite
 * @notice Pairs each representative AgentPay decode path against both runtimes:
 *   `*Old` → hand-tuned `agent-pay-contracts/src/lib/data/Pb.sol` snapshot.
 *   `*New` → currently-emitted `pb3-gen-sol` runtime.
 *
 *   Each pair builds the identical payload, then calls one variant. Gas is
 *   measured around just the decode call via `gasleft()`. Results are
 *   surfaced via Foundry's console-log precompile (see Console.sol);
 *   `forge test -vv` prints `<label> <payloadBytes> <gasUsed>` lines. See
 *   `plans/benchmarks/README.md` for the runner contract and reproduction
 *   steps.
 */
contract DecodeBenchTest is BenchBase {
    // Stable, recognizable test data shared by every benchmark below.
    bytes32 internal constant CHANNEL_ID = 0x1111111111111111111111111111111111111111111111111111111111111111;
    bytes32 internal constant PAY_ID_1 = 0x2222222222222222222222222222222222222222222222222222222222222222;
    bytes32 internal constant PAY_ID_2 = 0x3333333333333333333333333333333333333333333333333333333333333333;
    bytes32 internal constant HASH_LOCK = 0x4444444444444444444444444444444444444444444444444444444444444444;
    bytes32 internal constant NEXT_LIST_HASH = 0x5555555555555555555555555555555555555555555555555555555555555555;
    address internal constant SRC = address(0x1111111111111111111111111111111111111111);
    address internal constant DEST = address(0x2222222222222222222222222222222222222222);
    address internal constant PEER_FROM = address(0x3333333333333333333333333333333333333333);
    address internal constant PAY_RESOLVER = address(0x4444444444444444444444444444444444444444);
    address internal constant TOKEN = address(0x5555555555555555555555555555555555555555);
    address internal constant ACCOUNT_A = address(0x6666666666666666666666666666666666666666);
    address internal constant ACCOUNT_B = address(0x7777777777777777777777777777777777777777);

    // -------------------------------------------------------------------------
    // Path 1: ConditionalPay (3 conditions, mixed types)
    // -------------------------------------------------------------------------

    function _buildConditionalPay() internal pure returns (bytes memory) {
        Fixtures.Condition[] memory conds = new Fixtures.Condition[](3);
        conds[0] = Fixtures.condHashLock(HASH_LOCK);
        conds[1] = Fixtures.condDeployedBoolean(address(0xabcd), true);
        conds[2] = Fixtures.condDeployedNumeric(address(0xef01), 42);

        Fixtures.ConditionalPay memory pay = Fixtures.ConditionalPay({
            payTimestamp: 1700000000,
            src: SRC,
            dest: DEST,
            conditions: conds,
            logicType: 0, // BOOLEAN_AND
            maxAmount: 1_000_000,
            resolveDeadline: 17_000_000,
            resolveTimeout: 600,
            payResolver: PAY_RESOLVER
        });
        return Fixtures.encConditionalPay(pay);
    }

    function test_bench_decConditionalPay_old() public {
        bytes memory raw = _buildConditionalPay();
        uint256 g0 = gasleft();
        PbEntityOld.ConditionalPay memory pay = PbEntityOld.decConditionalPay(raw);
        uint256 g1 = gasleft();
        _measure("decConditionalPay_old", raw.length, g0, g1);

        // Sanity: decoded fields match.
        assertEq(pay.payTimestamp, 1700000000);
        assertEq(pay.src, SRC);
        assertEq(pay.dest, DEST);
        assertEq(pay.conditions.length, 3);
        assertEq(pay.payResolver, PAY_RESOLVER);
    }

    function test_bench_decConditionalPay_new() public {
        bytes memory raw = _buildConditionalPay();
        uint256 g0 = gasleft();
        PbEntityNew.ConditionalPay memory pay = PbEntityNew.decConditionalPay(raw);
        uint256 g1 = gasleft();
        _measure("decConditionalPay_new", raw.length, g0, g1);

        assertEq(pay.payTimestamp, 1700000000);
        assertEq(pay.src, SRC);
        assertEq(pay.dest, DEST);
        assertEq(pay.conditions.length, 3);
        assertEq(pay.payResolver, PAY_RESOLVER);
    }

    // -------------------------------------------------------------------------
    // Path 2: SimplexPaymentChannel with embedded PayIdList of 2 ids
    // -------------------------------------------------------------------------

    function _buildSimplexPaymentChannel() internal pure returns (bytes memory) {
        bytes32[] memory ids = new bytes32[](2);
        ids[0] = PAY_ID_1;
        ids[1] = PAY_ID_2;
        bytes memory payIdList = Fixtures.encPayIdList(ids, NEXT_LIST_HASH);

        Fixtures.SimplexState memory s = Fixtures.SimplexState({
            channelId: CHANNEL_ID,
            peerFrom: PEER_FROM,
            seqNum: 42,
            transferAmount: 12_345_678,
            pendingPayIds: payIdList,
            lastPayResolveDeadline: 17_000_000,
            totalPendingAmount: 100_000_000
        });
        return Fixtures.encSimplexPaymentChannel(s);
    }

    function test_bench_decSimplexPaymentChannel_old() public {
        bytes memory raw = _buildSimplexPaymentChannel();
        uint256 g0 = gasleft();
        PbEntityOld.SimplexPaymentChannel memory s = PbEntityOld.decSimplexPaymentChannel(raw);
        uint256 g1 = gasleft();
        _measure("decSimplexPaymentChannel_old", raw.length, g0, g1);

        assertEq(s.channelId, CHANNEL_ID);
        assertEq(s.peerFrom, PEER_FROM);
        assertEq(s.seqNum, 42);
        assertEq(s.totalPendingAmount, 100_000_000);
        assertEq(s.pendingPayIds.payIds.length, 2);
    }

    function test_bench_decSimplexPaymentChannel_new() public {
        bytes memory raw = _buildSimplexPaymentChannel();
        uint256 g0 = gasleft();
        PbEntityNew.SimplexPaymentChannel memory s = PbEntityNew.decSimplexPaymentChannel(raw);
        uint256 g1 = gasleft();
        _measure("decSimplexPaymentChannel_new", raw.length, g0, g1);

        assertEq(s.channelId, CHANNEL_ID);
        assertEq(s.peerFrom, PEER_FROM);
        assertEq(s.seqNum, 42);
        assertEq(s.totalPendingAmount, 100_000_000);
        assertEq(s.pendingPayIds.payIds.length, 2);
    }

    // -------------------------------------------------------------------------
    // Path 3: PaymentChannelInitializer (ETH, 2 peers)
    // -------------------------------------------------------------------------

    function _buildPaymentChannelInitializer() internal pure returns (bytes memory) {
        Fixtures.PaymentChannelInitializer memory init = Fixtures.PaymentChannelInitializer({
            tokenType: 1, // ETH
            tokenAddress: address(0),
            peers: [ACCOUNT_A, ACCOUNT_B],
            amounts: [uint256(1_000_000), uint256(2_000_000)],
            openDeadline: 17_000_000,
            disputeTimeout: 600,
            msgValueReceiver: 0
        });
        return Fixtures.encPaymentChannelInitializer(init);
    }

    function test_bench_decPaymentChannelInitializer_old() public {
        bytes memory raw = _buildPaymentChannelInitializer();
        uint256 g0 = gasleft();
        PbEntityOld.PaymentChannelInitializer memory init = PbEntityOld.decPaymentChannelInitializer(raw);
        uint256 g1 = gasleft();
        _measure("decPaymentChannelInitializer_old", raw.length, g0, g1);

        assertEq(init.openDeadline, 17_000_000);
        assertEq(init.disputeTimeout, 600);
        assertEq(init.initDistribution.distribution.length, 2);
    }

    function test_bench_decPaymentChannelInitializer_new() public {
        bytes memory raw = _buildPaymentChannelInitializer();
        uint256 g0 = gasleft();
        PbEntityNew.PaymentChannelInitializer memory init = PbEntityNew.decPaymentChannelInitializer(raw);
        uint256 g1 = gasleft();
        _measure("decPaymentChannelInitializer_new", raw.length, g0, g1);

        assertEq(init.openDeadline, 17_000_000);
        assertEq(init.disputeTimeout, 600);
        assertEq(init.initDistribution.distribution.length, 2);
    }

    // -------------------------------------------------------------------------
    // Path 4: CooperativeWithdrawInfo
    // -------------------------------------------------------------------------

    function _buildCooperativeWithdrawInfo() internal pure returns (bytes memory) {
        Fixtures.CooperativeWithdrawInfo memory w = Fixtures.CooperativeWithdrawInfo({
            channelId: CHANNEL_ID,
            seqNum: 7,
            withdrawAccount: ACCOUNT_A,
            withdrawAmount: 50_000,
            withdrawDeadline: 17_000_000,
            recipientChannelId: bytes32(0)
        });
        return Fixtures.encCooperativeWithdrawInfo(w);
    }

    function test_bench_decCooperativeWithdrawInfo_old() public {
        bytes memory raw = _buildCooperativeWithdrawInfo();
        uint256 g0 = gasleft();
        PbEntityOld.CooperativeWithdrawInfo memory w = PbEntityOld.decCooperativeWithdrawInfo(raw);
        uint256 g1 = gasleft();
        _measure("decCooperativeWithdrawInfo_old", raw.length, g0, g1);

        assertEq(w.channelId, CHANNEL_ID);
        assertEq(w.seqNum, 7);
        assertEq(w.withdraw.amt, 50_000);
    }

    function test_bench_decCooperativeWithdrawInfo_new() public {
        bytes memory raw = _buildCooperativeWithdrawInfo();
        uint256 g0 = gasleft();
        PbEntityNew.CooperativeWithdrawInfo memory w = PbEntityNew.decCooperativeWithdrawInfo(raw);
        uint256 g1 = gasleft();
        _measure("decCooperativeWithdrawInfo_new", raw.length, g0, g1);

        assertEq(w.channelId, CHANNEL_ID);
        assertEq(w.seqNum, 7);
        assertEq(w.withdraw.amt, 50_000);
    }

    // -------------------------------------------------------------------------
    // Path 5: CooperativeSettleInfo (2 settle accounts)
    // -------------------------------------------------------------------------

    function _buildCooperativeSettleInfo() internal pure returns (bytes memory) {
        Fixtures.CooperativeSettleInfo memory s = Fixtures.CooperativeSettleInfo({
            channelId: CHANNEL_ID,
            seqNum: 99,
            settleAccounts: [ACCOUNT_A, ACCOUNT_B],
            settleAmounts: [uint256(1_500_000), uint256(2_500_000)],
            settleDeadline: 17_000_000
        });
        return Fixtures.encCooperativeSettleInfo(s);
    }

    function test_bench_decCooperativeSettleInfo_old() public {
        bytes memory raw = _buildCooperativeSettleInfo();
        uint256 g0 = gasleft();
        PbEntityOld.CooperativeSettleInfo memory s = PbEntityOld.decCooperativeSettleInfo(raw);
        uint256 g1 = gasleft();
        _measure("decCooperativeSettleInfo_old", raw.length, g0, g1);

        assertEq(s.channelId, CHANNEL_ID);
        assertEq(s.seqNum, 99);
        assertEq(s.settleBalance.length, 2);
    }

    function test_bench_decCooperativeSettleInfo_new() public {
        bytes memory raw = _buildCooperativeSettleInfo();
        uint256 g0 = gasleft();
        PbEntityNew.CooperativeSettleInfo memory s = PbEntityNew.decCooperativeSettleInfo(raw);
        uint256 g1 = gasleft();
        _measure("decCooperativeSettleInfo_new", raw.length, g0, g1);

        assertEq(s.channelId, CHANNEL_ID);
        assertEq(s.seqNum, 99);
        assertEq(s.settleBalance.length, 2);
    }

    // -------------------------------------------------------------------------
    // Path 6: ResolvePayByConditionsRequest -> ConditionalPay (full flow)
    // -------------------------------------------------------------------------

    function _buildResolvePayRequest() internal pure returns (bytes memory) {
        bytes memory condPay = _buildConditionalPay();
        bytes[] memory preimages = new bytes[](3);
        preimages[0] = abi.encodePacked(uint256(0xaaaa));
        preimages[1] = abi.encodePacked(uint256(0xbbbb));
        preimages[2] = abi.encodePacked(uint256(0xcccc));
        return Fixtures.encResolvePayByConditionsRequest(condPay, preimages);
    }

    function test_bench_decResolvePayRequestThenConditionalPay_old() public {
        bytes memory raw = _buildResolvePayRequest();
        uint256 g0 = gasleft();
        PbChainOld.ResolvePayByConditionsRequest memory req = PbChainOld.decResolvePayByConditionsRequest(raw);
        PbEntityOld.ConditionalPay memory pay = PbEntityOld.decConditionalPay(req.condPay);
        uint256 g1 = gasleft();
        _measure("decResolvePayRequest+decConditionalPay_old", raw.length, g0, g1);

        assertEq(req.hashPreimages.length, 3);
        assertEq(pay.conditions.length, 3);
    }

    function test_bench_decResolvePayRequestThenConditionalPay_new() public {
        bytes memory raw = _buildResolvePayRequest();
        uint256 g0 = gasleft();
        PbChainNew.ResolvePayByConditionsRequest memory req = PbChainNew.decResolvePayByConditionsRequest(raw);
        PbEntityNew.ConditionalPay memory pay = PbEntityNew.decConditionalPay(req.condPay);
        uint256 g1 = gasleft();
        _measure("decResolvePayRequest+decConditionalPay_new", raw.length, g0, g1);

        assertEq(req.hashPreimages.length, 3);
        assertEq(pay.conditions.length, 3);
    }

    // -------------------------------------------------------------------------
    // Path 7: SignedSimplexState (wrapper) -> SimplexPaymentChannel
    // -------------------------------------------------------------------------

    function _buildSignedSimplexState() internal pure returns (bytes memory) {
        bytes memory simplex = _buildSimplexPaymentChannel();
        bytes[] memory sigs = new bytes[](2);
        sigs[0] = new bytes(65);
        sigs[1] = new bytes(65);
        for (uint256 i = 0; i < 65; i++) {
            sigs[0][i] = bytes1(uint8(0xaa));
            sigs[1][i] = bytes1(uint8(0xbb));
        }
        return Fixtures.encSignedSimplexState(simplex, sigs);
    }

    function test_bench_decSignedSimplexStateThenSimplex_old() public {
        bytes memory raw = _buildSignedSimplexState();
        uint256 g0 = gasleft();
        PbChainOld.SignedSimplexState memory wrap = PbChainOld.decSignedSimplexState(raw);
        PbEntityOld.SimplexPaymentChannel memory s = PbEntityOld.decSimplexPaymentChannel(wrap.simplexState);
        uint256 g1 = gasleft();
        _measure("decSignedSimplexState+decSimplex_old", raw.length, g0, g1);

        assertEq(wrap.sigs.length, 2);
        assertEq(s.channelId, CHANNEL_ID);
    }

    function test_bench_decSignedSimplexStateThenSimplex_new() public {
        bytes memory raw = _buildSignedSimplexState();
        uint256 g0 = gasleft();
        PbChainNew.SignedSimplexState memory wrap = PbChainNew.decSignedSimplexState(raw);
        PbEntityNew.SimplexPaymentChannel memory s = PbEntityNew.decSimplexPaymentChannel(wrap.simplexState);
        uint256 g1 = gasleft();
        _measure("decSignedSimplexState+decSimplex_new", raw.length, g0, g1);

        assertEq(wrap.sigs.length, 2);
        assertEq(s.channelId, CHANNEL_ID);
    }
}
