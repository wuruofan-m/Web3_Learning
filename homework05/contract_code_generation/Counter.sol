// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Counter {
    uint256 private count;
    address public owner;

    event CountChanged(uint256 newCount);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner can call this function");
        _;
    }

    constructor(uint256 initialCount) {
        count = initialCount;
        owner = msg.sender;
        emit CountChanged(count);
    }

    function increment() public {
        count += 1;
        emit CountChanged(count);
    }

    function decrement() public {
        require(count > 0, "Count cannot be negative");
        count -= 1;
        emit CountChanged(count);
    }

    function getCount() public view returns (uint256) {
        return count;
    }

    function reset() public onlyOwner {
        count = 0;
        emit CountChanged(count);
    }

    function transferOwnership(address newOwner) public onlyOwner {
        require(newOwner != address(0), "New owner is the zero address");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }
}
