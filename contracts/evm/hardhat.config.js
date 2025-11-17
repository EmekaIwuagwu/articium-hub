require("@nomicfoundation/hardhat-toolbox");
require("@nomicfoundation/hardhat-verify");
require("@openzeppelin/hardhat-upgrades");
require("hardhat-gas-reporter");
require("dotenv").config();

const PRIVATE_KEY = process.env.DEPLOYER_PRIVATE_KEY || "0x0000000000000000000000000000000000000000000000000000000000000001";
const ALCHEMY_API_KEY = process.env.ALCHEMY_API_KEY || "";
const INFURA_API_KEY = process.env.INFURA_API_KEY || "";
const NODEREAL_API_KEY = process.env.NODEREAL_API_KEY || "";

// Block explorer API keys
const POLYGONSCAN_API_KEY = process.env.POLYGONSCAN_API_KEY || "";
const BSCSCAN_API_KEY = process.env.BSCSCAN_API_KEY || "";
const SNOWTRACE_API_KEY = process.env.SNOWTRACE_API_KEY || "";
const ETHERSCAN_API_KEY = process.env.ETHERSCAN_API_KEY || "";

module.exports = {
  solidity: {
    version: "0.8.20",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200,
      },
      viaIR: true,
    },
  },

  networks: {
    // ===== TESTNETS =====

    // Polygon Amoy Testnet
    "polygon-amoy": {
      url: `https://polygon-amoy.g.alchemy.com/v2/${ALCHEMY_API_KEY}`,
      accounts: [PRIVATE_KEY],
      chainId: 80002,
      gasPrice: "auto",
      gas: "auto",
      timeout: 60000,
    },

    // BNB Smart Chain Testnet
    "bnb-testnet": {
      url: "https://data-seed-prebsc-1-s1.binance.org:8545/",
      accounts: [PRIVATE_KEY],
      chainId: 97,
      gasPrice: 10000000000, // 10 Gwei
      gas: 5000000,
      timeout: 60000,
    },

    // Avalanche Fuji Testnet
    "avalanche-fuji": {
      url: "https://api.avax-test.network/ext/bc/C/rpc",
      accounts: [PRIVATE_KEY],
      chainId: 43113,
      gasPrice: 25000000000, // 25 Gwei
      gas: 8000000,
      timeout: 60000,
    },

    // Ethereum Sepolia Testnet
    "ethereum-sepolia": {
      url: `https://sepolia.infura.io/v3/${INFURA_API_KEY}`,
      accounts: [PRIVATE_KEY],
      chainId: 11155111,
      gasPrice: "auto",
      gas: "auto",
      timeout: 60000,
    },

    // ===== MAINNETS =====

    // Polygon Mainnet
    "polygon-mainnet": {
      url: `https://polygon-mainnet.g.alchemy.com/v2/${ALCHEMY_API_KEY}`,
      accounts: [PRIVATE_KEY],
      chainId: 137,
      gasPrice: "auto",
      gas: "auto",
      timeout: 120000,
    },

    // BNB Smart Chain Mainnet
    "bnb-mainnet": {
      url: `https://bsc-mainnet.nodereal.io/v1/${NODEREAL_API_KEY}`,
      accounts: [PRIVATE_KEY],
      chainId: 56,
      gasPrice: 5000000000, // 5 Gwei
      gas: 5000000,
      timeout: 120000,
    },

    // Avalanche Mainnet
    "avalanche-mainnet": {
      url: `https://avalanche-mainnet.infura.io/v3/${INFURA_API_KEY}`,
      accounts: [PRIVATE_KEY],
      chainId: 43114,
      gasPrice: 25000000000, // 25 Gwei
      gas: 8000000,
      timeout: 120000,
    },

    // Ethereum Mainnet
    "ethereum-mainnet": {
      url: `https://mainnet.infura.io/v3/${INFURA_API_KEY}`,
      accounts: [PRIVATE_KEY],
      chainId: 1,
      gasPrice: "auto",
      gas: "auto",
      timeout: 120000,
    },

    // Local Hardhat Network
    hardhat: {
      chainId: 31337,
      forking: {
        enabled: false,
      },
    },

    // Local development
    localhost: {
      url: "http://127.0.0.1:8545",
      chainId: 31337,
    },
  },

  etherscan: {
    apiKey: {
      // Testnets
      polygonAmoy: POLYGONSCAN_API_KEY,
      bscTestnet: BSCSCAN_API_KEY,
      avalancheFujiTestnet: SNOWTRACE_API_KEY,
      sepolia: ETHERSCAN_API_KEY,

      // Mainnets
      polygon: POLYGONSCAN_API_KEY,
      bsc: BSCSCAN_API_KEY,
      avalanche: SNOWTRACE_API_KEY,
      mainnet: ETHERSCAN_API_KEY,
    },
    customChains: [
      {
        network: "polygonAmoy",
        chainId: 80002,
        urls: {
          apiURL: "https://api-amoy.polygonscan.com/api",
          browserURL: "https://amoy.polygonscan.com"
        }
      },
      {
        network: "avalancheFujiTestnet",
        chainId: 43113,
        urls: {
          apiURL: "https://api-testnet.snowtrace.io/api",
          browserURL: "https://testnet.snowtrace.io"
        }
      }
    ]
  },

  gasReporter: {
    enabled: process.env.REPORT_GAS === "true",
    currency: "USD",
    coinmarketcap: process.env.COINMARKETCAP_API_KEY,
    outputFile: "gas-report.txt",
    noColors: true,
  },

  paths: {
    sources: "./contracts",
    tests: "./test",
    cache: "./cache",
    artifacts: "./artifacts",
  },

  mocha: {
    timeout: 120000,
  },
};
