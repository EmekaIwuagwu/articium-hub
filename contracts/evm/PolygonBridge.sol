// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./BridgeBase.sol";

/**
 * @title PolygonBridge
 * @notice Bridge contract deployed on Polygon network (Amoy testnet / Mainnet)
 * @dev Inherits from BridgeBase and adds Polygon-specific functionality
 */
contract PolygonBridge is BridgeBase {
    /// @notice Polygon chain ID (80002 for Amoy, 137 for Mainnet)
    uint256 public constant POLYGON_CHAIN_ID = block.chainid;

    /// @notice Version of the bridge contract
    string public constant VERSION = "1.0.0";

    /**
     * @notice Initialize the Polygon bridge
     * @param _requiredSignatures Number of required validator signatures
     * @param _validators Array of validator addresses
     * @param _maxTransactionAmount Maximum transaction amount
     * @param _dailyLimit Daily transaction limit
     */
    function initialize(
        uint256 _requiredSignatures,
        address[] memory _validators,
        uint256 _maxTransactionAmount,
        uint256 _dailyLimit
    ) external initializer {
        string memory chainName = POLYGON_CHAIN_ID == 137 ? "Polygon" : "Polygon-Amoy";

        __BridgeBase_init(
            POLYGON_CHAIN_ID,
            chainName,
            _requiredSignatures,
            _validators,
            _maxTransactionAmount,
            _dailyLimit
        );
    }

    /**
     * @notice Get contract version
     */
    function version() external pure returns (string memory) {
        return VERSION;
    }

    /**
     * @notice Receive function to accept native MATIC
     */
    receive() external payable {}

    /**
     * @notice Fallback function
     */
    fallback() external payable {}
}
