// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title Proto
 * @notice Minimal protobuf wire-format encoder used by Foundry tests to construct
 *  the binary messages that AgentPay contracts decode (`PbEntity` / `PbChain`).
 *  Produces output that round-trips through the existing decoders.
 *
 * @dev Wire format primer:
 *  - tag byte = `(fieldNum << 3) | wireType`
 *  - wire types used here: `0 = Varint`, `2 = LengthDelim`
 *  - varint = 7 bits per byte, MSB=1 means "more bytes follow"
 *  - `bytes` / embedded message: tag, then varint length, then payload
 *
 * @dev Encoding policy: scalar fields with default value (zero) are omitted;
 *  length-delimited fields are emitted whenever `length > 0`. Callers that need
 *  to embed a zero-valued bytes field with `length == 0` should not call the
 *  helper at all (which is what protobuf would do anyway).
 */
library Proto {
    uint256 internal constant WIRE_VARINT = 0;
    uint256 internal constant WIRE_LEN = 2;

    // -------------------------------------------------------------------------
    // Primitives
    // -------------------------------------------------------------------------

    /// @dev Encode an unsigned varint.
    function varint(uint256 _v) internal pure returns (bytes memory out) {
        // Worst case: 10 bytes (uint64 max takes 10 bytes; we never go bigger).
        bytes memory tmp = new bytes(10);
        uint256 i = 0;
        while (_v >= 0x80) {
            tmp[i] = bytes1(uint8((_v & 0x7F) | 0x80));
            _v >>= 7;
            i++;
        }
        tmp[i] = bytes1(uint8(_v));
        i++;
        out = new bytes(i);
        for (uint256 j = 0; j < i; j++) {
            out[j] = tmp[j];
        }
    }

    /// @dev Encode a wire-format key (`(fieldNum << 3) | wireType`) as a varint.
    function key(uint256 _fieldNum, uint256 _wireType) internal pure returns (bytes memory) {
        return varint((_fieldNum << 3) | _wireType);
    }

    /// @dev Encode a length-prefixed payload: `varint(payload.length) | payload`.
    function lenPrefixed(bytes memory _payload) internal pure returns (bytes memory) {
        return abi.encodePacked(varint(_payload.length), _payload);
    }

    /// @dev Concatenate two byte arrays. Solidity 0.8 doesn't have a native operator.
    function cat(bytes memory _a, bytes memory _b) internal pure returns (bytes memory) {
        return abi.encodePacked(_a, _b);
    }

    /// @dev Concatenate three byte arrays.
    function cat3(bytes memory _a, bytes memory _b, bytes memory _c) internal pure returns (bytes memory) {
        return abi.encodePacked(_a, _b, _c);
    }

    // -------------------------------------------------------------------------
    // Per-field-type helpers
    // -------------------------------------------------------------------------

    /// @dev Encode a varint field. Skips entirely if `_v == 0` (proto3 default).
    function uintField(uint256 _fieldNum, uint256 _v) internal pure returns (bytes memory) {
        if (_v == 0) return "";
        return cat(key(_fieldNum, WIRE_VARINT), varint(_v));
    }

    /// @dev Encode an enum field. Same as `uintField` (also skipped at zero).
    function enumField(uint256 _fieldNum, uint256 _v) internal pure returns (bytes memory) {
        return uintField(_fieldNum, _v);
    }

    /// @dev Encode a length-delimited field (bytes / embedded message). Skips if empty.
    function bytesField(uint256 _fieldNum, bytes memory _v) internal pure returns (bytes memory) {
        if (_v.length == 0) return "";
        return cat(key(_fieldNum, WIRE_LEN), lenPrefixed(_v));
    }

    /// @dev Encode an `address`-typed `bytes` field (always 20 bytes when present).
    function addressField(uint256 _fieldNum, address _v) internal pure returns (bytes memory) {
        if (_v == address(0)) return "";
        return bytesField(_fieldNum, abi.encodePacked(_v));
    }

    /// @dev Encode a `bytes32`-typed `bytes` field (always 32 bytes when present).
    function bytes32Field(uint256 _fieldNum, bytes32 _v) internal pure returns (bytes memory) {
        if (_v == bytes32(0)) return "";
        return bytesField(_fieldNum, abi.encodePacked(_v));
    }

    /**
     * @dev Encode a `uint256` value as the length-delimited bytes form used by the
     *  AgentPay protobuf messages (soltype = "uint256"). Stored big-endian, with
     *  leading zero bytes stripped except that zero itself is encoded as one
     *  zero byte.
     */
    function uint256ToBytes(uint256 _v) internal pure returns (bytes memory out) {
        if (_v == 0) {
            out = new bytes(1);
            return out;
        }
        // Find the highest non-zero byte.
        uint256 n = 0;
        uint256 tmp = _v;
        while (tmp != 0) {
            n++;
            tmp >>= 8;
        }
        out = new bytes(n);
        for (uint256 i = 0; i < n; i++) {
            out[n - 1 - i] = bytes1(uint8(_v >> (i * 8)));
        }
    }

    /// @dev Encode a `uint256` field that uses the "bytes (soltype=uint256)" wire shape.
    ///  Always emitted when non-zero; for zero the JS encoder still emits a single 0
    ///  byte if the field was explicitly set, but our policy is "omit zero" to match
    ///  proto3 default semantics. Use `bytesField` directly with a forced single-zero
    ///  byte payload if you need to express "explicit zero".
    function uint256Field(uint256 _fieldNum, uint256 _v) internal pure returns (bytes memory) {
        if (_v == 0) return "";
        return bytesField(_fieldNum, uint256ToBytes(_v));
    }

    /// @dev Encode a `uint256` field with an *explicit zero* payload (single 0 byte).
    ///  Some messages depend on this — e.g. `transferToPeer.amt = 0` in checkpoint
    ///  states needs to be present so the decoder reads zero rather than missing.
    function uint256FieldExplicit(uint256 _fieldNum, uint256 _v) internal pure returns (bytes memory) {
        return bytesField(_fieldNum, uint256ToBytes(_v));
    }

    /// @dev Encode a repeated length-delimited field. Each element gets its own tag.
    ///  (Repeated `bytes` and repeated embedded messages aren't packed in proto3.)
    function repeatedBytesField(uint256 _fieldNum, bytes[] memory _items) internal pure returns (bytes memory out) {
        for (uint256 i = 0; i < _items.length; i++) {
            // Allow empty entries — protobuf encodes `bytes ""` as length=0 payload.
            out = cat(out, cat(key(_fieldNum, WIRE_LEN), lenPrefixed(_items[i])));
        }
    }

    /// @dev Encode a repeated `bytes32` field. Each element gets its own tag.
    function repeatedBytes32Field(uint256 _fieldNum, bytes32[] memory _items) internal pure returns (bytes memory out) {
        for (uint256 i = 0; i < _items.length; i++) {
            out = cat(out, cat(key(_fieldNum, WIRE_LEN), lenPrefixed(abi.encodePacked(_items[i]))));
        }
    }
}
