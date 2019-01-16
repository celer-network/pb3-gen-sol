pragma solidity >=0.4.24;

import "./lib/mytest.sol";
import "./lib/a.sol";
import "./lib/b.sol";

contract TestMain {
    event Msg1Part1(
        uint32 f1,
        uint64 f2,
        bool f3,
        bytes f4,
        string f5,
        uint f6Len,
        uint32[] f6,
        uint f7Len,
        uint64[] f7
    );

    event Msg1Part2(
        uint f8Len,
        bool[] f8,
        uint f9Len,
        bytes f9_0,
        bytes f9_1,
        uint f10Len,
        string f10_0,
        string f10_1
    );

    event Msg2Part1(
        uint8 num,
        address addr,
        uint256 amt,
        bytes32 h,
        uint numsLen,
        uint8[] nums,
        uint addrsLen,
        address[] addrs
    );

    event Msg2Part2(
        uint amtsLen,
        uint256[] amts,
        uint hsLen,
        bytes32[] hs,
        uint ts,
        uint tssLen,
        uint[] tss
    );

    event Msg4Info(
        uint enum1,
        uint enum2,
        uint enumsLen,
        uint[] enums
    );

    event Debug();

    function testMsg1(bytes memory raw) public {
        mytest.Msg1 memory m = mytest.decMsg1(raw);

        emit Msg1Part1(
            m.f1, 
            m.f2, 
            m.f3,
            m.f4,
            m.f5,
            m.f6.length,
            m.f6, 
            m.f7.length,
            m.f7
        );

        emit Msg1Part2(
            m.f8.length,
            m.f8, 
            m.f9.length,
            m.f9[0],
            m.f9[1],
            m.f10.length,
            m.f10[0],
            m.f10[1]
        );
    }

    function testMsg2(bytes memory raw) public {
        mytest.Msg2 memory m = mytest.decMsg2(raw);

        emit Msg2Part1(
            m.num,
            m.addr,
            m.amt,
            m.hash,
            m.nums.length,
            m.nums,
            m.addrs.length,
            m.addrs
        );

        emit Msg2Part2(
            m.amts.length,
            m.amts,
            m.hashes.length,
            m.hashes,
            m.ts,
            m.tss.length,
            m.tss
        );
    }

    function testMsg3(bytes memory raw) public {
        mytest.Msg3 memory m = mytest.decMsg3(raw);

        // emit events of m1
        emit Msg1Part1(
            m.m1.f1, 
            m.m1.f2, 
            m.m1.f3,
            m.m1.f4,
            m.m1.f5,
            m.m1.f6.length, 
            m.m1.f6, 
            m.m1.f7.length,
            m.m1.f7
        );

        emit Msg1Part2( 
            m.m1.f8.length, 
            m.m1.f8, 
            m.m1.f9.length,
            m.m1.f9[0],
            m.m1.f9[1],
            m.m1.f10.length,
            m.m1.f10[0],
            m.m1.f10[1]
        );

        // emit events of m2
        emit Msg2Part1(
            m.m2.num,
            m.m2.addr,
            m.m2.amt,
            m.m2.hash,
            m.m2.nums.length,
            m.m2.nums,
            m.m2.addrs.length,
            m.m2.addrs
        );

        emit Msg2Part2(
            m.m2.amts.length,
            m.m2.amts,
            m.m2.hashes.length,
            m.m2.hashes,
            m.m2.ts,
            m.m2.tss.length,
            m.m2.tss
        );

        // emit events of m1s[0]
        emit Msg1Part1(
            m.m1s[0].f1, 
            m.m1s[0].f2, 
            m.m1s[0].f3,
            m.m1s[0].f4,
            m.m1s[0].f5,
            m.m1s[0].f6.length, 
            m.m1s[0].f6, 
            m.m1s[0].f7.length,
            m.m1s[0].f7
        );

        emit Msg1Part2(
            m.m1s[0].f8.length, 
            m.m1s[0].f8, 
            m.m1s[0].f9.length, 
            m.m1s[0].f9[0],
            m.m1s[0].f9[1],
            m.m1s[0].f10.length, 
            m.m1s[0].f10[0],
            m.m1s[0].f10[1]
        );

        // emit events of m1s[1]
        emit Msg1Part1(
            m.m1s[1].f1, 
            m.m1s[1].f2, 
            m.m1s[1].f3,
            m.m1s[1].f4,
            m.m1s[1].f5,
            m.m1s[1].f6.length, 
            m.m1s[1].f6, 
            m.m1s[1].f7.length, 
            m.m1s[1].f7
        );

        emit Msg1Part2( 
            m.m1s[1].f8.length, 
            m.m1s[1].f8, 
            m.m1s[1].f9.length, 
            m.m1s[1].f9[0],
            m.m1s[1].f9[1],
            m.m1s[1].f10.length, 
            m.m1s[1].f10[0],
            m.m1s[1].f10[1]
        );

        // emit events of m2s[0]
        emit Msg2Part1(
            m.m2s[0].num,
            m.m2s[0].addr,
            m.m2s[0].amt,
            m.m2s[0].hash,
            m.m2s[0].nums.length,
            m.m2s[0].nums,
            m.m2s[0].addrs.length,
            m.m2s[0].addrs
        );

        emit Msg2Part2(
            m.m2s[0].amts.length,
            m.m2s[0].amts,
            m.m2s[0].hashes.length,
            m.m2s[0].hashes,
            m.m2s[0].ts,
            m.m2s[0].tss.length,
            m.m2s[0].tss
        );

        // emit events of m2s[1]
        emit Msg2Part1(
            m.m2s[1].num,
            m.m2s[1].addr,
            m.m2s[1].amt,
            m.m2s[1].hash,
            m.m2s[1].nums.length,
            m.m2s[1].nums,
            m.m2s[1].addrs.length,
            m.m2s[1].addrs
        );

        emit Msg2Part2(
            m.m2s[1].amts.length,
            m.m2s[1].amts,
            m.m2s[1].hashes.length,
            m.m2s[1].hashes,
            m.m2s[1].ts,
            m.m2s[1].tss.length,
            m.m2s[1].tss
        );
    }

    function testMsg4(bytes memory raw) public {
        mytest.Msg4 memory m = mytest.decMsg4(raw);

        uint[] memory uintEnums = new uint[](m.enums.length);
        for (uint i = 0; i < uintEnums.length; i++) { uintEnums[i] = uint(m.enums[i]); }

        emit Msg4Info(
            uint(m.enum1),
            uint(m.enum2),
            m.enums.length,
            uintEnums
        );
    }
    event DecodedB (
        uint64 i,
        uint alistlen,
        a.MyEnum e,
        uint elistlen
    );
    function testImport(bytes memory raw) public {
        b.B memory m = b.decB(raw);
        emit DecodedB(m.i.f1, m.alist.length, m.e, m.elist.length);
    }
}