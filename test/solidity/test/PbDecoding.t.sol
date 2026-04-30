// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import {PbA} from "../src/lib/PbA.sol";
import {PbB} from "../src/lib/PbB.sol";
import {PbMytest} from "../src/lib/PbMytest.sol";

import {TestBase} from "./utils/TestBase.sol";

contract PbDecodingTest is TestBase {
    address private constant ADDR_A = address(uint160(0x000002030405060708090a0b0c0d0e0f1011121314));
    address private constant ADDR_B = address(uint160(0x000102030405060708090a0b0c0d0e0f1011121314));
    address private constant ADDR_C = address(uint160(0x000b0c0d0e0f10111213140102030405060708090a));
    address private constant ADDR_D = address(uint160(0x00ff02030405060708090a0b0c0d0e0f1011121314));
    address private constant ADDR_E = address(uint160(0x00000c0d0e0f10111213140102030405060708090a));

    bytes32 private constant HASH_1 = 0x0101010101010101010101010101010101010101010101010101010101010101;
    bytes32 private constant HASH_2 = 0x0202020202020202020202020202020202020202020202020202020202020202;
    bytes32 private constant HASH_3 = 0x0303030303030303030303030303030303030303030303030303030303030303;

    function testDecodesMsg1() external {
        PbMytest.Msg1 memory m = PbMytest.decMsg1(_loadFixture("msg1.pb"));

        assertEq(uint256(m.f1), 1);
        assertEq(uint256(m.f2), 12);
        assertEq(m.f3, true);
        assertEq(m.f4, hex"01020304");
        assertEq(m.f5, "abc");

        assertEq(m.f6.length, 3);
        assertEq(uint256(m.f6[0]), 1);
        assertEq(uint256(m.f6[1]), 2);
        assertEq(uint256(m.f6[2]), 3);

        assertEq(m.f7.length, 3);
        assertEq(uint256(m.f7[0]), 11);
        assertEq(uint256(m.f7[1]), 12);
        assertEq(uint256(m.f7[2]), 13);

        assertEq(m.f8.length, 3);
        assertEq(m.f8[0], true);
        assertEq(m.f8[1], false);
        assertEq(m.f8[2], true);

        assertEq(m.f9.length, 2);
        assertEq(m.f9[0], hex"0102");
        assertEq(m.f9[1], hex"0304");

        assertEq(m.f10.length, 2);
        assertEq(m.f10[0], "ab");
        assertEq(m.f10[1], "cd");
    }

    function testDecodesMsg1LargeNumbers() external {
        PbMytest.Msg1 memory m = PbMytest.decMsg1(_loadFixture("msg1_large_number.pb"));

        assertEq(uint256(m.f1), type(uint32).max);
        assertEq(uint256(m.f2), type(uint64).max);
        assertEq(m.f6.length, 3);
        assertEq(uint256(m.f6[0]), type(uint32).max);
        assertEq(uint256(m.f6[1]), type(uint32).max - 1);
        assertEq(uint256(m.f6[2]), type(uint32).max - 2);
        assertEq(m.f7.length, 3);
        assertEq(uint256(m.f7[0]), type(uint64).max);
        assertEq(uint256(m.f7[1]), type(uint64).max - 1);
        assertEq(uint256(m.f7[2]), type(uint64).max - 2);
    }

    function testDecodesMsg2() external {
        PbMytest.Msg2 memory m = PbMytest.decMsg2(_loadFixture("msg2.pb"));

        _assertMsg2(m, 12, ADDR_A, ADDR_B, HASH_1, 123456, 12345678, 87654321);
    }

    function testDecodesMsg2LargeNumbers() external {
        PbMytest.Msg2 memory m = PbMytest.decMsg2(_loadFixture("msg2_large_number.pb"));

        assertEq(m.ts, type(uint64).max);
        assertEq(m.tss.length, 3);
        assertEq(m.tss[0], type(uint64).max);
        assertEq(m.tss[1], type(uint64).max - 1);
        assertEq(m.tss[2], type(uint64).max - 2);
    }

    function testAllowsEmptyTopLevelMessages() external pure {
        PbMytest.Msg1 memory m = PbMytest.decMsg1(hex"");

        assertEq(uint256(m.f1), 0);
        assertEq(uint256(m.f2), 0);
        assertEq(m.f3, false);
        assertEq(m.f4, bytes(""));
        assertEq(m.f5, "");
        assertEq(m.f6.length, 0);
        assertEq(m.f7.length, 0);
        assertEq(m.f8.length, 0);
    }

    function testAllowsEmptyNestedMessages() external pure {
        PbMytest.Msg3 memory m = PbMytest.decMsg3(hex"0a00");

        assertEq(uint256(m.m1.f1), 0);
        assertEq(m.m1.f5, "");
        assertEq(m.m1s.length, 0);
        assertEq(m.m2s.length, 0);
    }

    function testAllowsEmptyRepeatedNestedMessages() external pure {
        PbMytest.Msg3 memory m = PbMytest.decMsg3(hex"1a00");

        assertEq(m.m1s.length, 1);
        assertEq(uint256(m.m1s[0].f1), 0);
        assertEq(m.m1s[0].f5, "");
    }

    function testSkipsUnknownHighTags() external pure {
        PbMytest.Msg1 memory m = PbMytest.decMsg1(hex"a00600");

        assertEq(uint256(m.f1), 0);
        assertEq(m.f9.length, 0);
        assertEq(m.f10.length, 0);
    }

    function testRejectsTruncatedLengthDelimitedFields() external {
        vm.expectRevert();
        this.decodeMsg1(hex"22030102");
    }

    function testRejectsTruncatedVarints() external {
        vm.expectRevert();
        this.decodeMsg1(hex"08");
    }

    function testRejectsMalformedPackedVarints() external {
        vm.expectRevert();
        this.decodeMsg1(hex"320180");
    }

    function testRejectsInvalidWireTypes() external {
        vm.expectRevert();
        this.decodeMsg1(hex"0e");

        vm.expectRevert();
        this.decodeMsg1(hex"0f");
    }

    function testSkipsUnknownFixedWireTypes() external pure {
        // Unknown tag 11, wire Fixed64 (0x59 = (11<<3)|1). 8 payload bytes
        // are skipped, then a known tag 1 (varint, value 7) is decoded.
        PbMytest.Msg1 memory m = PbMytest.decMsg1(hex"5901020304050607080807");
        assertEq(uint256(m.f1), 7);

        // Same with wire Fixed32 (0x5d = (11<<3)|5). 4 payload bytes skipped,
        // then tag 1 = 7.
        m = PbMytest.decMsg1(hex"5d010203040807");
        assertEq(uint256(m.f1), 7);
    }

    function testRejectsTruncatedFixedWireTypes() external {
        // Unknown tag 11 wire Fixed64 (0x59) but only 4 payload bytes follow.
        vm.expectRevert();
        this.decodeMsg1(hex"5901020304");

        // Unknown tag 11 wire Fixed32 (0x5d) but only 2 payload bytes follow.
        vm.expectRevert();
        this.decodeMsg1(hex"5d0102");
    }

    function testMsg2RejectsWrongAddressLength() external {
        vm.expectRevert();
        this.decodeMsg2(_loadFixture("msg2_wrong_addr.pb"));
    }

    function testMsg2RejectsWrongBytes32Length() external {
        vm.expectRevert();
        this.decodeMsg2(_loadFixture("msg2_wrong_bytes32.pb"));
    }

    function testMsg2RejectsOversizedUint256Bytes() external {
        bytes memory raw = new bytes(35);
        raw[0] = 0x22;
        raw[1] = 0x21;
        for (uint256 i = 0; i < 33; i++) {
            raw[i + 2] = 0x01;
        }

        vm.expectRevert();
        this.decodeMsg2(raw);
    }

    function testDecodesMsg3() external {
        PbMytest.Msg3 memory m = PbMytest.decMsg3(_loadFixture("msg3.pb"));

        assertEq(uint256(m.m1.f1), 1);
        assertEq(uint256(m.m1.f2), 12);
        assertEq(m.m1.f5, "abc");

        _assertMsg2(m.m2, 12, ADDR_A, ADDR_B, HASH_1, 12345678, 12345678, 12345678);

        assertEq(m.m1s.length, 2);
        assertEq(uint256(m.m1s[0].f1), 2);
        assertEq(uint256(m.m1s[1].f1), 3);

        assertEq(m.m2s.length, 2);
        _assertMsg2(m.m2s[0], 111, ADDR_E, ADDR_C, HASH_2, 12345, 123456, 1234567);
        _assertMsg2(m.m2s[1], 222, ADDR_A, ADDR_D, HASH_3, 54321, 654321, 7654321);
    }

    function testDecodesMsg4() external {
        PbMytest.Msg4 memory m = PbMytest.decMsg4(_loadFixture("msg4.pb"));

        assertEq(uint256(m.enum1), 1);
        assertEq(uint256(m.enum2), 2);
        assertEq(m.enums.length, 3);
        assertEq(uint256(m.enums[0]), 0);
        assertEq(uint256(m.enums[1]), 1);
        assertEq(uint256(m.enums[2]), 2);
    }

    function testDecodesImportedMessages() external {
        PbB.B memory m = PbB.decB(_loadFixture("b.pb"));

        assertEq(uint256(m.i.f1), 1);
        assertEq(m.alist.length, 2);
        assertEq(uint256(m.alist[0].f1), 2);
        assertEq(uint256(m.alist[1].f1), 3);
        assertEq(uint256(m.e), uint256(PbA.MyEnum.E0));
        assertEq(m.elist.length, 2);
        assertEq(uint256(m.elist[0]), uint256(PbA.MyEnum.E0));
        assertEq(uint256(m.elist[1]), uint256(PbA.MyEnum.E1));
    }

    function decodeMsg1(bytes memory raw) external pure returns (PbMytest.Msg1 memory) {
        return PbMytest.decMsg1(raw);
    }

    function decodeMsg2(bytes memory raw) external pure returns (PbMytest.Msg2 memory) {
        return PbMytest.decMsg2(raw);
    }

    function _assertMsg2(
        PbMytest.Msg2 memory m,
        uint256 expectedNum,
        address expectedAddr,
        address expectedAddrPayable,
        bytes32 expectedHash,
        uint256 expectedTs,
        uint256 expectedTss0,
        uint256 expectedTss1
    ) internal pure {
        assertEq(uint256(m.num), expectedNum);
        assertEq(m.addr, expectedAddr);
        assertEq(address(m.addrPayable), expectedAddrPayable);
        assertEq(m.amt, 117967114);
        assertEq(m.hash, expectedHash);

        assertEq(m.nums.length, 3);
        assertEq(uint256(m.nums[0]), 12);
        assertEq(uint256(m.nums[1]), 34);
        assertEq(uint256(m.nums[2]), 56);

        assertEq(m.addrs.length, 3);
        assertEq(m.addrs[0], ADDR_A);
        assertEq(m.addrs[1], ADDR_C);
        assertEq(m.addrs[2], ADDR_D);

        assertEq(m.addrPayables.length, 3);
        assertEq(address(m.addrPayables[0]), ADDR_B);
        assertEq(address(m.addrPayables[1]), ADDR_C);
        assertEq(address(m.addrPayables[2]), ADDR_D);

        assertEq(m.amts.length, 3);
        assertEq(m.amts[0], 1);
        assertEq(m.amts[1], 257);
        assertEq(m.amts[2], 65793);

        assertEq(m.hashes.length, 3);
        assertEq(m.hashes[0], HASH_1);
        assertEq(m.hashes[1], HASH_2);
        assertEq(m.hashes[2], HASH_3);

        assertEq(m.ts, expectedTs);
        assertEq(m.tss.length, 2);
        assertEq(m.tss[0], expectedTss0);
        assertEq(m.tss[1], expectedTss1);
    }

    function _loadFixture(string memory name) internal returns (bytes memory) {
        return vm.readFileBinary(string.concat("../fixtures/bin/", name));
    }
}
