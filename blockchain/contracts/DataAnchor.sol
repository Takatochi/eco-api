// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract DataAnchor {
    event HashAnchored(bytes32 indexed dataHash, address indexed sender, uint256 blockNumber, uint256 timestamp);

    mapping(bytes32 => uint256) public anchoredAtBlock;

    function anchor(bytes32 dataHash) external {
        if (anchoredAtBlock[dataHash] == 0) {
            anchoredAtBlock[dataHash] = block.number;
        }
        emit HashAnchored(dataHash, msg.sender, block.number, block.timestamp);
    }

    function isAnchored(bytes32 dataHash) external view returns (bool) {
        return anchoredAtBlock[dataHash] != 0;
    }
}
