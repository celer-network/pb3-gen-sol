// SPDX-License-Identifier: MIT
pragma solidity >=0.8.0;

/**
 * @title Console
 * @notice Minimal `console.log` shim that targets Foundry's built-in console
 *  precompile. Avoids pulling in forge-std for one helper.
 */
library Console {
    address private constant CONSOLE_ADDRESS = 0x000000000000000000636F6e736F6c652e6c6f67;

    function log(string memory key, uint256 v1, uint256 v2) internal view {
        bytes memory payload = abi.encodeWithSignature("log(string,uint256,uint256)", key, v1, v2);
        (bool ok,) = CONSOLE_ADDRESS.staticcall(payload);
        ok; // discard: console precompile is a no-op outside Foundry
    }
}
