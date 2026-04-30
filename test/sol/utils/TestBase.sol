// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

interface Vm {
    function readFileBinary(string calldata path) external returns (bytes memory);
    function expectRevert() external;
}

abstract contract TestBase {
    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm internal constant vm = Vm(VM_ADDRESS);

    function assertEq(uint256 actual, uint256 expected) internal pure {
        require(actual == expected, "assertEq(uint256)");
    }

    function assertEq(address actual, address expected) internal pure {
        require(actual == expected, "assertEq(address)");
    }

    function assertEq(bool actual, bool expected) internal pure {
        require(actual == expected, "assertEq(bool)");
    }

    function assertEq(bytes32 actual, bytes32 expected) internal pure {
        require(actual == expected, "assertEq(bytes32)");
    }

    function assertEq(bytes memory actual, bytes memory expected) internal pure {
        require(keccak256(actual) == keccak256(expected), "assertEq(bytes)");
    }

    function assertEq(string memory actual, string memory expected) internal pure {
        require(keccak256(bytes(actual)) == keccak256(bytes(expected)), "assertEq(string)");
    }
}
