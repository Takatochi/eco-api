const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();
  const Factory = await ethers.getContractFactory("DataAnchor");
  const contract = await Factory.deploy();
  await contract.waitForDeployment();

  const address = await contract.getAddress();
  console.log("DEPLOYER=", deployer.address);
  console.log("CONTRACT=", address);
  console.log(`BLOCKCHAIN_CONTRACT_ADDRESS=${address}`);
  console.log(`BLOCKCHAIN_FROM_ADDRESS=${deployer.address}`);
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
