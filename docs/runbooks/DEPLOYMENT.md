# Deployment Runbook

This runbook guides you through deploying the Metabridge Hub bridge protocol to testnet and mainnet environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Infrastructure Setup](#infrastructure-setup)
3. [Smart Contract Deployment](#smart-contract-deployment)
4. [Backend Services Deployment](#backend-services-deployment)
5. [Post-Deployment Verification](#post-deployment-verification)
6. [Rollback Procedures](#rollback-procedures)

## Prerequisites

### Required Tools

- Docker and Docker Compose 20.10+
- Go 1.21+
- Rust 1.70+ with `wasm32-unknown-unknown` target
- Node.js 18+ and Yarn (for Solana Anchor)
- NEAR CLI
- Solana CLI
- Hardhat or Foundry (for EVM contracts)
- kubectl (for Kubernetes deployments)
- PostgreSQL client
- NATS CLI

### Required Access

- AWS/GCP/Azure cloud provider account
- RPC endpoints for all supported chains
- Validator private keys (HSM or KMS recommended)
- Database credentials
- NATS cluster credentials
- Docker registry access
- Domain and SSL certificates

## Infrastructure Setup

### 1. Database Setup

```bash
# Create PostgreSQL database
createdb metabridge_testnet  # or metabridge_mainnet

# Run migrations
cd /path/to/metabridge-hub
go run cmd/migrator/main.go --config config/config.testnet.yaml
```

### 2. NATS Cluster Setup

```bash
# Deploy NATS with JetStream
docker run -d \
  --name nats-server \
  -p 4222:4222 \
  -p 8222:8222 \
  nats:latest \
  -js \
  -m 8222

# Verify NATS is running
nats-cli server ping
```

### 3. Redis Cache Setup

```bash
# Deploy Redis
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# Test connection
redis-cli ping
```

### 4. Monitoring Stack

```bash
# Deploy Prometheus + Grafana
cd deployments/docker
docker-compose -f docker-compose.monitoring.yaml up -d

# Access Grafana at http://localhost:3000
# Default credentials: admin/admin
```

## Smart Contract Deployment

### EVM Chains (Ethereum, Polygon, BNB, Avalanche)

#### Testnet Deployment

```bash
cd contracts/evm

# Configure environment
cp .env.example .env.testnet
# Edit .env.testnet with:
# - PRIVATE_KEY (deployer private key)
# - RPC_URL_POLYGON_AMOY
# - RPC_URL_BNB_TESTNET
# - RPC_URL_AVALANCHE_FUJI
# - RPC_URL_ETHEREUM_SEPOLIA

# Deploy to Polygon Amoy
npx hardhat run scripts/deploy.ts --network polygon-amoy

# Deploy to BNB Testnet
npx hardhat run scripts/deploy.ts --network bnb-testnet

# Deploy to Avalanche Fuji
npx hardhat run scripts/deploy.ts --network avalanche-fuji

# Deploy to Ethereum Sepolia
npx hardhat run scripts/deploy.ts --network ethereum-sepolia

# Save contract addresses
# Update config/config.testnet.yaml with deployed addresses
```

#### Mainnet Deployment

⚠️ **WARNING**: Mainnet deployment requires multi-signature approval and audit

```bash
# Use hardware wallet or multi-sig for deployment
# Recommended: Use Gnosis Safe or similar multi-sig

# Deploy to Polygon Mainnet
npx hardhat run scripts/deploy.ts --network polygon-mainnet

# Verify contracts on block explorers
npx hardhat verify --network polygon-mainnet <CONTRACT_ADDRESS> <CONSTRUCTOR_ARGS>
```

### Solana Bridge

#### Testnet (Devnet) Deployment

```bash
cd contracts/solana

# Configure Solana CLI for devnet
solana config set --url https://api.devnet.solana.com

# Create program keypair (or use existing)
solana-keygen new -o ./target/deploy/solana_bridge-keypair.json

# Fund the deployer account
solana airdrop 2

# Build program
anchor build

# Deploy to devnet
anchor deploy --provider.cluster devnet

# Initialize bridge
anchor run initialize-devnet
```

#### Mainnet Deployment

```bash
# Switch to mainnet
solana config set --url https://api.mainnet-beta.solana.com

# Deploy with proper authority
anchor deploy --provider.cluster mainnet --program-keypair ./mainnet-keypair.json

# Initialize with mainnet validators
# Use multi-sig for upgrade authority
```

### NEAR Bridge

#### Testnet Deployment

```bash
cd contracts/near

# Build contract
./build.sh

# Deploy to testnet
near deploy \
  --accountId metabridge.testnet \
  --wasmFile ./res/near_bridge.wasm

# Initialize contract
near call metabridge.testnet new \
  '{
    "owner": "admin.testnet",
    "validators": ["ed25519:..."],
    "required_signatures": 2
  }' \
  --accountId admin.testnet
```

#### Mainnet Deployment

```bash
# Deploy to mainnet account
near deploy \
  --accountId bridge.metabridge.near \
  --wasmFile ./res/near_bridge.wasm \
  --networkId mainnet

# Initialize with mainnet configuration
near call bridge.metabridge.near new \
  '{
    "owner": "owner.metabridge.near",
    "validators": [...],
    "required_signatures": 3
  }' \
  --accountId owner.metabridge.near \
  --networkId mainnet
```

## Backend Services Deployment

### Using Docker Compose (Simple Deployment)

```bash
cd deployments/docker

# Testnet deployment
docker-compose -f docker-compose.testnet.yaml up -d

# Verify services are running
docker-compose ps

# View logs
docker-compose logs -f
```

### Using Kubernetes (Production Deployment)

```bash
cd deployments/kubernetes

# Create namespace
kubectl create namespace metabridge-mainnet

# Create secrets
kubectl create secret generic bridge-secrets \
  --from-literal=db-password='<DB_PASSWORD>' \
  --from-literal=validator-key='<VALIDATOR_KEY>' \
  -n metabridge-mainnet

# Deploy services
kubectl apply -f configmap.yaml -n metabridge-mainnet
kubectl apply -f api-deployment.yaml -n metabridge-mainnet
kubectl apply -f listener-deployment.yaml -n metabridge-mainnet
kubectl apply -f relayer-deployment.yaml -n metabridge-mainnet

# Verify deployments
kubectl get pods -n metabridge-mainnet
kubectl get services -n metabridge-mainnet
```

### Service-by-Service Deployment

#### 1. API Service

```bash
# Build binary
go build -o bin/api cmd/api/main.go

# Run with systemd
sudo systemctl start metabridge-api

# Verify
curl http://localhost:8080/health
```

#### 2. Listener Service

```bash
# Build binary
go build -o bin/listener cmd/listener/main.go

# Run
sudo systemctl start metabridge-listener

# Check logs
sudo journalctl -u metabridge-listener -f
```

#### 3. Relayer Service

```bash
# Build binary
go build -o bin/relayer cmd/relayer/main.go

# Run
sudo systemctl start metabridge-relayer

# Monitor
sudo journalctl -u metabridge-relayer -f
```

## Post-Deployment Verification

### 1. Health Checks

```bash
# API health
curl http://localhost:8080/health

# Check all chains are connected
curl http://localhost:8080/v1/chains

# Verify database connection
psql -h localhost -U metabridge -d metabridge_testnet -c "SELECT COUNT(*) FROM chains;"
```

### 2. Smoke Tests

```bash
# Test token lock on testnet
curl -X POST http://localhost:8080/v1/bridge/token \
  -H "Content-Type: application/json" \
  -d '{
    "source_chain": "polygon",
    "destination_chain": "avalanche",
    "token_address": "0x...",
    "amount": "1000000",
    "sender": "0x...",
    "recipient": "0x..."
  }'

# Verify message was created
curl http://localhost:8080/v1/messages/<MESSAGE_ID>
```

### 3. Monitor Metrics

```bash
# Check Prometheus metrics
curl http://localhost:9090/metrics

# Verify key metrics
curl http://localhost:9090/api/v1/query?query=bridge_messages_total
curl http://localhost:9090/api/v1/query?query=chain_health
```

### 4. End-to-End Test

```bash
# Run E2E test suite
cd tests/e2e
go test -v ./... -config ../../config/config.testnet.yaml
```

## Rollback Procedures

### Rolling Back Services

```bash
# Docker Compose rollback
cd deployments/docker
docker-compose down
docker-compose -f docker-compose.testnet.yaml up -d --build --force-recreate

# Kubernetes rollback
kubectl rollout undo deployment/api -n metabridge-mainnet
kubectl rollout undo deployment/listener -n metabridge-mainnet
kubectl rollout undo deployment/relayer -n metabridge-mainnet

# Verify rollback
kubectl rollout status deployment/api -n metabridge-mainnet
```

### Rolling Back Smart Contracts

⚠️ **EVM Contracts**: Upgradeable proxies only

```bash
# Rollback to previous implementation
npx hardhat run scripts/upgrade.ts --network polygon-mainnet --previous

# Verify
npx hardhat verify <PREVIOUS_IMPLEMENTATION>
```

⚠️ **Solana**: Program upgrade required

```bash
# Upgrade to previous version
anchor upgrade ./target/previous/solana_bridge.so \
  --program-id <PROGRAM_ID> \
  --provider.cluster mainnet
```

⚠️ **NEAR**: Code update via DAO or owner

```bash
# Redeploy previous version
near deploy --accountId bridge.metabridge.near \
  --wasmFile ./previous/near_bridge.wasm \
  --networkId mainnet
```

### Database Rollback

```bash
# Restore from backup
pg_restore -h localhost -U metabridge -d metabridge_mainnet \
  --clean --if-exists \
  backups/metabridge_mainnet_TIMESTAMP.dump

# Verify restoration
psql -h localhost -U metabridge -d metabridge_mainnet \
  -c "SELECT MAX(created_at) FROM messages;"
```

## Security Checklist

Before going live, verify:

- [ ] All validator keys stored in HSM/KMS
- [ ] Database encrypted at rest
- [ ] TLS/SSL enabled for all services
- [ ] Firewall rules configured
- [ ] Rate limiting enabled
- [ ] Monitoring and alerting configured
- [ ] Backup procedures tested
- [ ] Incident response plan documented
- [ ] Multi-signature required for admin operations
- [ ] Security audit completed
- [ ] Bug bounty program active

## Support Contacts

- **Technical Lead**: tech-lead@metabridge.io
- **DevOps**: devops@metabridge.io
- **Security**: security@metabridge.io
- **On-Call**: +1-XXX-XXX-XXXX (PagerDuty)
