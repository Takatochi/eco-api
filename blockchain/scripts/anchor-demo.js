const { ethers } = require("ethers");

async function main() {
  const rpcUrl = process.env.BLOCKCHAIN_RPC_URL || "http://127.0.0.1:8545";
  const contractAddress = process.env.BLOCKCHAIN_CONTRACT_ADDRESS;
  const privateKey = process.env.BLOCKCHAIN_PRIVATE_KEY;
  const dataHash = process.env.DATA_HASH;

  if (!contractAddress || !privateKey || !dataHash) {
    throw new Error("Set BLOCKCHAIN_CONTRACT_ADDRESS, BLOCKCHAIN_PRIVATE_KEY, DATA_HASH");
  }

  const abi = [
    "function anchor(bytes32 dataHash)",
    "function anchoredAtBlock(bytes32) view returns (uint256)",
  ];

  const provider = new ethers.JsonRpcProvider(rpcUrl);
  const wallet = new ethers.Wallet(privateKey, provider);
  const contract = new ethers.Contract(contractAddress, abi, wallet);

  const tx = await contract.anchor(dataHash);
  const receipt = await tx.wait();
  console.log("txHash:", receipt.hash, "block:", receipt.blockNumber);
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
