// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Proto} from "./Proto.sol";

/**
 * @title Fixtures
 * @notice Solidity builders for the AgentPay protobuf messages used in Foundry
 *  tests. Encoding is done via {Proto}; output round-trips through the existing
 *  `PbEntity` / `PbChain` decoders.
 *
 * @dev Encoders are organized by source proto file:
 *  - `entity.proto` messages: AccountAmtPair, TokenInfo, TokenDistribution,
 *    TokenTransfer, SimplexPaymentChannel, PayIdList, TransferFunction,
 *    ConditionalPay, CondPayResult, VouchedCondPayResult, Condition,
 *    CooperativeWithdrawInfo, PaymentChannelInitializer, CooperativeSettleInfo,
 *    ChannelMigrationInfo.
 *  - `chain.proto` messages: OpenChannelRequest, CooperativeWithdrawRequest,
 *    CooperativeSettleRequest, ResolvePayByConditionsRequest,
 *    SignedSimplexState, SignedSimplexStateArray, ChannelMigrationRequest.
 */
library Fixtures {
    using Proto for bytes;

    // -------------------------------------------------------------------------
    // entity.proto
    // -------------------------------------------------------------------------

    /// @notice Encode `entity.AccountAmtPair { account: address, amt: uint256 }`.
    function encAccountAmtPair(address _account, uint256 _amt) internal pure returns (bytes memory) {
        return Proto.cat(Proto.addressField(1, _account), Proto.uint256FieldExplicit(2, _amt));
    }

    /// @notice Encode `entity.TokenInfo { tokenType: enum, tokenAddress: address }`.
    function encTokenInfo(uint256 _tokenType, address _tokenAddress) internal pure returns (bytes memory) {
        return Proto.cat(Proto.enumField(1, _tokenType), Proto.addressField(2, _tokenAddress));
    }

    /// @notice Encode `entity.TokenDistribution`.
    function encTokenDistribution(
        uint256 _tokenType,
        address _tokenAddress,
        address[2] memory _accounts,
        uint256[2] memory _amounts
    ) internal pure returns (bytes memory out) {
        // field 1: TokenInfo
        out = Proto.bytesField(1, encTokenInfo(_tokenType, _tokenAddress));
        // field 2: repeated AccountAmtPair
        out = Proto.cat(out, Proto.bytesField(2, encAccountAmtPair(_accounts[0], _amounts[0])));
        out = Proto.cat(out, Proto.bytesField(2, encAccountAmtPair(_accounts[1], _amounts[1])));
    }

    /// @notice Encode `entity.TokenTransfer { token: TokenInfo (optional), receiver: AccountAmtPair }`.
    /// @dev If `_tokenType == 0` and `_tokenAddress == 0`, the token field is omitted.
    function encTokenTransfer(uint256 _tokenType, address _tokenAddress, address _receiverAccount, uint256 _receiverAmt)
        internal
        pure
        returns (bytes memory out)
    {
        bytes memory tokenInfo = encTokenInfo(_tokenType, _tokenAddress);
        if (tokenInfo.length > 0) {
            out = Proto.bytesField(1, tokenInfo);
        }
        out = Proto.cat(out, Proto.bytesField(2, encAccountAmtPair(_receiverAccount, _receiverAmt)));
    }

    /**
     * @notice Encode `entity.PayIdList`.
     * @param _payIds Pay ids in this list segment.
     * @param _nextListHash Hash of the next list segment, or `bytes32(0)` for the tail.
     */
    function encPayIdList(bytes32[] memory _payIds, bytes32 _nextListHash) internal pure returns (bytes memory out) {
        out = Proto.repeatedBytes32Field(1, _payIds);
        if (_nextListHash != bytes32(0)) {
            out = Proto.cat(out, Proto.bytes32Field(2, _nextListHash));
        }
    }

    /// @notice Encode `entity.SimplexPaymentChannel`.
    struct SimplexState {
        bytes32 channelId;
        address peerFrom;
        uint256 seqNum;
        uint256 transferAmount;
        bytes pendingPayIds; // pre-encoded PayIdList (use {encPayIdList})
        uint256 lastPayResolveDeadline;
        uint256 totalPendingAmount;
    }

    function encSimplexPaymentChannel(SimplexState memory _s) internal pure returns (bytes memory out) {
        out = Proto.bytes32Field(1, _s.channelId);
        out = Proto.cat(out, Proto.addressField(2, _s.peerFrom));
        out = Proto.cat(out, Proto.uintField(3, _s.seqNum));
        // Field 4: transfer_to_peer (TokenTransfer with no token field, just receiver=0/amt).
        bytes memory transferToPeer = encTokenTransfer(0, address(0), address(0), _s.transferAmount);
        if (transferToPeer.length > 0) {
            out = Proto.cat(out, Proto.bytesField(4, transferToPeer));
        }
        // Field 5: pending_pay_ids (PayIdList — pre-encoded by caller).
        if (_s.pendingPayIds.length > 0) {
            out = Proto.cat(out, Proto.bytesField(5, _s.pendingPayIds));
        }
        out = Proto.cat(out, Proto.uintField(6, _s.lastPayResolveDeadline));
        out = Proto.cat(out, Proto.uint256Field(7, _s.totalPendingAmount));
    }

    /**
     * @notice Encode `entity.Condition`.
     * @dev `_conditionType`:
     *  - `0` → HASH_LOCK; uses `_hashLock`.
     *  - `1` → DEPLOYED_CONTRACT; uses `_deployedAddress` and `_argsQueryOutcome`.
     *  - `2` → VIRTUAL_CONTRACT; uses `_virtualAddr`, `_argsQueryFinalization`,
     *    and `_argsQueryOutcome`.
     */
    struct Condition {
        uint256 conditionType;
        bytes32 hashLock;
        address deployedAddress;
        bytes32 virtualAddr;
        bytes argsQueryFinalization;
        bytes argsQueryOutcome;
    }

    function encCondition(Condition memory _c) internal pure returns (bytes memory out) {
        out = Proto.enumField(1, _c.conditionType);
        out = Proto.cat(out, Proto.bytes32Field(2, _c.hashLock));
        out = Proto.cat(out, Proto.addressField(3, _c.deployedAddress));
        out = Proto.cat(out, Proto.bytes32Field(4, _c.virtualAddr));
        out = Proto.cat(out, Proto.bytesField(5, _c.argsQueryFinalization));
        out = Proto.cat(out, Proto.bytesField(6, _c.argsQueryOutcome));
    }

    /// @notice Build a HASH_LOCK condition.
    function condHashLock(bytes32 _hashLock) internal pure returns (Condition memory c) {
        c.conditionType = 0;
        c.hashLock = _hashLock;
    }

    /// @notice Build a DEPLOYED_CONTRACT condition with one-byte boolean argsQueryOutcome.
    function condDeployedBoolean(address _addr, bool _outcome) internal pure returns (Condition memory c) {
        c.conditionType = 1;
        c.deployedAddress = _addr;
        c.argsQueryOutcome = abi.encodePacked(_outcome ? bytes1(0x01) : bytes1(0x00));
    }

    /// @notice Build a DEPLOYED_CONTRACT condition with one-byte numeric argsQueryOutcome.
    function condDeployedNumeric(address _addr, uint8 _amount) internal pure returns (Condition memory c) {
        c.conditionType = 1;
        c.deployedAddress = _addr;
        c.argsQueryOutcome = abi.encodePacked(_amount);
    }

    /// @notice Build a VIRTUAL_CONTRACT condition.
    function condVirtual(bytes32 _virtAddr, bytes memory _argsOutcome) internal pure returns (Condition memory c) {
        c.conditionType = 2;
        c.virtualAddr = _virtAddr;
        c.argsQueryOutcome = _argsOutcome;
    }

    /// @notice Encode `entity.TransferFunction`.
    function encTransferFunction(uint256 _logicType, uint256 _maxAmount) internal pure returns (bytes memory out) {
        out = Proto.enumField(1, _logicType);
        out = Proto.cat(out, Proto.bytesField(2, encTokenTransfer(0, address(0), address(0), _maxAmount)));
    }

    /// @notice Encode `entity.ConditionalPay`.
    struct ConditionalPay {
        uint256 payTimestamp;
        address src;
        address dest;
        Condition[] conditions;
        uint256 logicType;
        uint256 maxAmount;
        uint256 resolveDeadline;
        uint256 resolveTimeout;
        address payResolver;
    }

    function encConditionalPay(ConditionalPay memory _p) internal pure returns (bytes memory out) {
        out = Proto.uintField(1, _p.payTimestamp);
        out = Proto.cat(out, Proto.addressField(2, _p.src));
        out = Proto.cat(out, Proto.addressField(3, _p.dest));
        for (uint256 i = 0; i < _p.conditions.length; i++) {
            out = Proto.cat(out, Proto.bytesField(4, encCondition(_p.conditions[i])));
        }
        out = Proto.cat(out, Proto.bytesField(5, encTransferFunction(_p.logicType, _p.maxAmount)));
        out = Proto.cat(out, Proto.uintField(6, _p.resolveDeadline));
        out = Proto.cat(out, Proto.uintField(7, _p.resolveTimeout));
        out = Proto.cat(out, Proto.addressField(8, _p.payResolver));
    }

    /// @notice Encode `entity.CondPayResult { condPay: bytes, amount: uint256 }`.
    function encCondPayResult(bytes memory _condPay, uint256 _amount) internal pure returns (bytes memory out) {
        out = Proto.bytesField(1, _condPay);
        out = Proto.cat(out, Proto.uint256FieldExplicit(2, _amount));
    }

    /// @notice Encode `entity.VouchedCondPayResult { condPayResult, sigOfSrc, sigOfDest }`.
    function encVouchedCondPayResult(bytes memory _condPayResult, bytes memory _sigSrc, bytes memory _sigDest)
        internal
        pure
        returns (bytes memory out)
    {
        out = Proto.bytesField(1, _condPayResult);
        out = Proto.cat(out, Proto.bytesField(2, _sigSrc));
        out = Proto.cat(out, Proto.bytesField(3, _sigDest));
    }

    /// @notice Encode `entity.CooperativeWithdrawInfo`.
    struct CooperativeWithdrawInfo {
        bytes32 channelId;
        uint256 seqNum;
        address withdrawAccount;
        uint256 withdrawAmount;
        uint256 withdrawDeadline;
        bytes32 recipientChannelId;
    }

    function encCooperativeWithdrawInfo(CooperativeWithdrawInfo memory _w) internal pure returns (bytes memory out) {
        out = Proto.bytes32Field(1, _w.channelId);
        out = Proto.cat(out, Proto.uintField(2, _w.seqNum));
        out = Proto.cat(out, Proto.bytesField(3, encAccountAmtPair(_w.withdrawAccount, _w.withdrawAmount)));
        out = Proto.cat(out, Proto.uintField(4, _w.withdrawDeadline));
        out = Proto.cat(out, Proto.bytes32Field(5, _w.recipientChannelId));
    }

    /// @notice Encode `entity.PaymentChannelInitializer`.
    struct PaymentChannelInitializer {
        uint256 tokenType; // 1 = ETH, 2 = ERC20
        address tokenAddress; // 0 for ETH
        address[2] peers; // ascending order
        uint256[2] amounts;
        uint256 openDeadline;
        uint256 disputeTimeout;
        uint256 msgValueReceiver; // peer index
    }

    function encPaymentChannelInitializer(PaymentChannelInitializer memory _p)
        internal
        pure
        returns (bytes memory out)
    {
        out = Proto.bytesField(1, encTokenDistribution(_p.tokenType, _p.tokenAddress, _p.peers, _p.amounts));
        out = Proto.cat(out, Proto.uintField(2, _p.openDeadline));
        out = Proto.cat(out, Proto.uintField(3, _p.disputeTimeout));
        out = Proto.cat(out, Proto.uintField(4, _p.msgValueReceiver));
    }

    /// @notice Encode `entity.CooperativeSettleInfo`.
    struct CooperativeSettleInfo {
        bytes32 channelId;
        uint256 seqNum;
        address[2] settleAccounts;
        uint256[2] settleAmounts;
        uint256 settleDeadline;
    }

    function encCooperativeSettleInfo(CooperativeSettleInfo memory _s) internal pure returns (bytes memory out) {
        out = Proto.bytes32Field(1, _s.channelId);
        out = Proto.cat(out, Proto.uintField(2, _s.seqNum));
        out = Proto.cat(out, Proto.bytesField(3, encAccountAmtPair(_s.settleAccounts[0], _s.settleAmounts[0])));
        out = Proto.cat(out, Proto.bytesField(3, encAccountAmtPair(_s.settleAccounts[1], _s.settleAmounts[1])));
        out = Proto.cat(out, Proto.uintField(4, _s.settleDeadline));
    }

    /// @notice Encode `entity.ChannelMigrationInfo`.
    struct ChannelMigrationInfo {
        bytes32 channelId;
        address fromLedger;
        address toLedger;
        uint256 migrationDeadline;
    }

    function encChannelMigrationInfo(ChannelMigrationInfo memory _m) internal pure returns (bytes memory out) {
        out = Proto.bytes32Field(1, _m.channelId);
        out = Proto.cat(out, Proto.addressField(2, _m.fromLedger));
        out = Proto.cat(out, Proto.addressField(3, _m.toLedger));
        out = Proto.cat(out, Proto.uintField(4, _m.migrationDeadline));
    }

    // -------------------------------------------------------------------------
    // chain.proto
    // -------------------------------------------------------------------------

    /// @notice Wrap a body in a `(bodyBytes, sigs)` pair: used by all `*Request`
    ///  messages and `SignedSimplexState`. Each emits field-1 = body, field-2 = sigs.
    function encBodyAndSigs(bytes memory _body, bytes[] memory _sigs) internal pure returns (bytes memory out) {
        out = Proto.bytesField(1, _body);
        out = Proto.cat(out, Proto.repeatedBytesField(2, _sigs));
    }

    /// @notice Encode `chain.OpenChannelRequest`.
    function encOpenChannelRequest(bytes memory _initializer, bytes[] memory _sigs)
        internal
        pure
        returns (bytes memory)
    {
        return encBodyAndSigs(_initializer, _sigs);
    }

    /// @notice Encode `chain.CooperativeWithdrawRequest`.
    function encCooperativeWithdrawRequest(bytes memory _info, bytes[] memory _sigs)
        internal
        pure
        returns (bytes memory)
    {
        return encBodyAndSigs(_info, _sigs);
    }

    /// @notice Encode `chain.CooperativeSettleRequest`.
    function encCooperativeSettleRequest(bytes memory _info, bytes[] memory _sigs)
        internal
        pure
        returns (bytes memory)
    {
        return encBodyAndSigs(_info, _sigs);
    }

    /// @notice Encode `chain.ChannelMigrationRequest`.
    function encChannelMigrationRequest(bytes memory _info, bytes[] memory _sigs) internal pure returns (bytes memory) {
        return encBodyAndSigs(_info, _sigs);
    }

    /// @notice Encode `chain.ResolvePayByConditionsRequest`.
    function encResolvePayByConditionsRequest(bytes memory _condPay, bytes[] memory _hashPreimages)
        internal
        pure
        returns (bytes memory out)
    {
        out = Proto.bytesField(1, _condPay);
        out = Proto.cat(out, Proto.repeatedBytesField(2, _hashPreimages));
    }

    /// @notice Encode `chain.SignedSimplexState`.
    function encSignedSimplexState(bytes memory _simplexState, bytes[] memory _sigs)
        internal
        pure
        returns (bytes memory)
    {
        return encBodyAndSigs(_simplexState, _sigs);
    }

    /// @notice Encode `chain.SignedSimplexStateArray`.
    function encSignedSimplexStateArray(bytes[] memory _signedSimplexStates) internal pure returns (bytes memory out) {
        for (uint256 i = 0; i < _signedSimplexStates.length; i++) {
            out = Proto.cat(out, Proto.bytesField(1, _signedSimplexStates[i]));
        }
    }
}
