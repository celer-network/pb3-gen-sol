// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import {BenchBase} from "./BenchBase.sol";
import {PbMytest} from "../../src/lib/PbMytest.sol";

interface FsVm {
    function readFileBinary(string calldata path) external returns (bytes memory);
}

/**
 * @title Stress benchmark suite
 * @notice Exercises the scratch-array (single-pass over-allocate + shrink)
 *  strategy on shapes that the AgentPay paths in `Decode.t.sol` do not
 *  cover:
 *
 *  - `decMsg2`: four scratch-eligible repeated fields decoded in the same
 *    message (`address`, `address payable`, `uint256`-bytes, `bytes32`),
 *    each with its own per-element-type upper bound.
 *  - `decMsg3`: nested `repeated Msg1` + `repeated Msg2` (reference types
 *    that fall back to `cntTags`) plus inner Msg2's multi-scratch shape on
 *    each element. Stresses the interaction between the two strategies.
 *
 *  No `Old` paired runtime here — `test.proto` has no legacy snapshot.
 *  The point is to monitor the absolute gas cost on stress shapes so any
 *  future regression on multi-scratch decoders shows up in CI.
 */
contract StressBenchTest is BenchBase {
    address private constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    FsVm private constant fsVm = FsVm(VM_ADDRESS);

    function _load(string memory name) internal returns (bytes memory) {
        return fsVm.readFileBinary(string.concat("../fixtures/bin/", name));
    }

    // 351-byte payload, four scratch-eligible repeated fields each holding
    // 3 elements. The over-allocation upper bounds at this payload size are:
    //   addrs / addrPayables (`address`, min wire 22): 351 / 22 = 15 slots
    //   amts (`uint256`, min wire 2):                  351 / 2  = 175 slots
    //   hashes (`bytes32`, min wire 34):               351 / 34 = 10 slots
    // Wasted-slot count is significant for `amts`; this is the path where
    // a future regression on the per-element-type minWireSize would show.
    function test_bench_decMsg2_multiScratch() public {
        bytes memory raw = _load("msg2.pb");
        uint256 g0 = gasleft();
        PbMytest.Msg2 memory m = PbMytest.decMsg2(raw);
        uint256 g1 = gasleft();
        _measure("decMsg2_multiScratch", raw.length, g0, g1);

        require(m.addrs.length == 3, "addrs");
        require(m.addrPayables.length == 3, "addrPayables");
        require(m.amts.length == 3, "amts");
        require(m.hashes.length == 3, "hashes");
    }

    // 1210-byte payload, nested. Outer `repeated Msg1` and `repeated Msg2`
    // are reference types and fall back to `cntTags`. Each inner Msg2
    // decode pays its own multi-scratch over-allocation (4 arrays sized
    // from the inner Msg2 payload, not the outer 1210 bytes — important
    // sanity check that we are not over-allocating from the wrong scope).
    function test_bench_decMsg3_nested() public {
        bytes memory raw = _load("msg3.pb");
        uint256 g0 = gasleft();
        PbMytest.Msg3 memory m = PbMytest.decMsg3(raw);
        uint256 g1 = gasleft();
        _measure("decMsg3_nested", raw.length, g0, g1);

        require(m.m1s.length == 2, "m1s");
        require(m.m2s.length == 2, "m2s");
    }
}
