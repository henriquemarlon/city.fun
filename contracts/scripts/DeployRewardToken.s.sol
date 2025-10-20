// SPDX-License-Identifier: MIT

pragma solidity ^0.8.27;

import {Script} from "forge-std-1.11.0/src/Script.sol";
import {console} from "forge-std-1.11.0/src/console.sol";
import {RewardToken} from "../src/token/RewardToken.sol";

contract DeployRewardToken is Script {
    RewardToken public token;

    function run() external {
        console.log("Starting deployment on chain id:", block.chainid);

        vm.startBroadcast();
        console.log("Deploying Reward Token...");
        address initialOwner = vm.parseAddress(vm.prompt("Initial owner"));
        token = new RewardToken{salt: bytes32(keccak256("1596"))}(initialOwner);
        console.log("Reward token deployed to:", address(token));

        vm.stopBroadcast();

        _saveDeploymentInfo();

        console.log("Tokens deployment completed!");
    }

    function _saveDeploymentInfo() internal {
        string memory deploymentInfo = string.concat(
            '{"rewardToken":{',
            '"chainId":',
            vm.toString(block.chainid),
            ",",
            '"timestamp":',
            vm.toString(block.timestamp),
            ",",
            '"contracts":{',
            '"rewardToken":"',
            vm.toString(address(token)),
            '"',
            "}",
            "}}"
        );
        vm.writeJson(deploymentInfo, "./deployments/rewardToken.json");
    }
}