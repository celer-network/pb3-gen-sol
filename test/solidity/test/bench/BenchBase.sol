// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

import {Console} from "./Console.sol";

abstract contract BenchBase {
    function _measure(string memory label, uint256 payloadBytes, uint256 startGas, uint256 endGas) internal view {
        Console.log(label, payloadBytes, startGas - endGas);
    }

    function assertEq(uint256 actual, uint256 expected) internal pure {
        require(actual == expected, "assertEq(uint256)");
    }

    function assertEq(address actual, address expected) internal pure {
        require(actual == expected, "assertEq(address)");
    }

    function assertEq(bytes32 actual, bytes32 expected) internal pure {
        require(actual == expected, "assertEq(bytes32)");
    }

    function assertEq(bytes memory actual, bytes memory expected) internal pure {
        require(keccak256(actual) == keccak256(expected), "assertEq(bytes)");
    }
}
