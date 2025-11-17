const hre = require("hardhat");

async function main() {
  console.log("========================================");
  console.log("  Deploy Bridge to ALL Testnets");
  console.log("========================================");
  console.log("");

  const networks = [
    { name: "polygon-amoy", chainId: 80002 },
    { name: "bnb-testnet", chainId: 97 },
    { name: "avalanche-fuji", chainId: 43113 },
    { name: "ethereum-sepolia", chainId: 11155111 },
  ];

  const deploymentResults = [];

  for (const network of networks) {
    console.log(`\n${"=".repeat(60)}`);
    console.log(`Deploying to ${network.name}...`);
    console.log("=".repeat(60));

    try {
      // Run deployment for this network
      await hre.run("run", {
        script: "scripts/deploy.js",
        network: network.name,
      });

      deploymentResults.push({
        network: network.name,
        chainId: network.chainId,
        status: "✅ Success",
      });
    } catch (error) {
      console.error(`❌ Failed to deploy to ${network.name}:`, error.message);
      deploymentResults.push({
        network: network.name,
        chainId: network.chainId,
        status: "❌ Failed",
        error: error.message,
      });
    }

    // Wait between deployments
    console.log("\nWaiting 5 seconds before next deployment...");
    await new Promise((resolve) => setTimeout(resolve, 5000));
  }

  // Print summary
  console.log("\n" + "=".repeat(60));
  console.log("  DEPLOYMENT SUMMARY");
  console.log("=".repeat(60));
  console.log("");

  deploymentResults.forEach((result) => {
    console.log(`${result.network} (${result.chainId}): ${result.status}`);
    if (result.error) {
      console.log(`  Error: ${result.error}`);
    }
  });

  console.log("");
  console.log("✅ All testnet deployments completed!");
  console.log("");
  console.log("📋 Next steps:");
  console.log("1. Check deployments/ directory for contract addresses");
  console.log("2. Update config/config.testnet.yaml with all addresses");
  console.log("3. Verify contracts on block explorers");
  console.log("4. Test cross-chain transfers");
  console.log("");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
