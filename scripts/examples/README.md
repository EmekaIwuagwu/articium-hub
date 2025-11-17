# Cross-Chain Transfer Examples

This directory contains example scripts for performing cross-chain transfers using the Metabridge Engine.

## Prerequisites

- Bridge API running at `http://localhost:8080`
- `jq` installed (for JSON parsing)
- `curl` installed
- Tokens on source chain

## Available Examples

### 1. Polygon to Avalanche

**Script**: `polygon-to-avalanche.sh`

Transfers WMATIC from Polygon Amoy to Avalanche Fuji.

```bash
chmod +x polygon-to-avalanche.sh
./polygon-to-avalanche.sh
```

**Parameters**:
- Source: Polygon Amoy
- Destination: Avalanche Fuji
- Token: WMATIC
- Amount: 1 token

### 2. BNB to Ethereum

**Script**: `bnb-to-ethereum.sh`

Transfers WBNB from BNB Testnet to Ethereum Sepolia.

```bash
chmod +x bnb-to-ethereum.sh
./bnb-to-ethereum.sh
```

**Parameters**:
- Source: BNB Testnet
- Destination: Ethereum Sepolia
- Token: WBNB
- Amount: 0.1 token

### 3. Node.js Script

**Script**: `../cross-chain-transfer.js`

Full-featured JavaScript example with monitoring.

```bash
# Install dependencies
npm install axios

# Run with default config
node ../cross-chain-transfer.js

# Or customize
SOURCE_CHAIN=polygon-amoy \
DEST_CHAIN=avalanche-fuji \
AMOUNT=1000000000000000000 \
node ../cross-chain-transfer.js
```

## Customizing Examples

### Change Source/Destination

Edit the script variables:

```bash
SOURCE_CHAIN="bnb-testnet"
DEST_CHAIN="polygon-amoy"
```

Supported chains:
- `polygon-amoy`
- `bnb-testnet`
- `avalanche-fuji`
- `ethereum-sepolia`

### Change Token Address

```bash
TOKEN_ADDRESS="0xYOUR_TOKEN_ADDRESS"
```

Common testnet tokens:
- **Polygon Amoy WMATIC**: `0x0000000000000000000000000000000000001010`
- **BNB Testnet WBNB**: `0xae13d989daC2f0dEbFf460aC112a837C89BAa7cd`
- **Avalanche Fuji WAVAX**: `0xd00ae08403B9bbb9124bB305C09058E32C39A48c`
- **Sepolia WETH**: `0xfFf9976782d46CC05630D1f6eBAb18b2324d6B14`

### Change Amount

Amount in wei (18 decimals):

```bash
# 1 token
AMOUNT="1000000000000000000"

# 0.1 token
AMOUNT="100000000000000000"

# 0.01 token
AMOUNT="10000000000000000"
```

### Change Sender/Recipient

```bash
SENDER="0xYOUR_SENDER_ADDRESS"
RECIPIENT="0xYOUR_RECIPIENT_ADDRESS"
```

## Expected Output

### Successful Transfer

```
========================================
  Polygon -> Avalanche Transfer
========================================

✅ Bridge is healthy

Initiating transfer...
  From: polygon-amoy
  To: avalanche-fuji
  Amount: 1000000000000000000 (1 token)

✅ Transfer initiated!
  Message ID: abc123...

Monitoring transfer...
(This may take several minutes)

  Status: pending (1/60)
  Status: validated (2/60)
  Status: processing (3/60)
🎉 Transfer completed!
{
  "id": "abc123...",
  "status": "completed",
  "destination_tx_hash": "0xdef456..."
}
```

### Failed Transfer

```
❌ Transfer failed
{
  "error": "insufficient funds",
  "details": "sender has insufficient balance"
}
```

## Monitoring Transfer

To check transfer status manually:

```bash
MESSAGE_ID="your-message-id"
curl http://localhost:8080/v1/messages/$MESSAGE_ID | jq '.'
```

## Transaction Lifecycle

1. **Pending**: Message created, waiting for validators
2. **Validated**: Validators signed the message
3. **Processing**: Relayer broadcasting to destination
4. **Completed**: Transaction confirmed on destination
5. **Failed**: Error occurred (check logs)

## Typical Times

| Route | Average Time | Confirmations |
|-------|-------------|---------------|
| Polygon → Avalanche | 3-5 min | 128 + 10 blocks |
| BNB → Ethereum | 2-4 min | 15 + 32 blocks |
| Avalanche → Polygon | 2-4 min | 10 + 128 blocks |
| Ethereum → BNB | 4-6 min | 32 + 15 blocks |

## Troubleshooting

### "Bridge is not healthy"

```bash
# Check API
curl http://localhost:8080/health

# Restart services
./stop-testnet.sh
./deploy-testnet.sh
```

### "Transfer failed: insufficient funds"

1. Check source address has tokens
2. Check source address has gas tokens
3. Verify token address is correct

### "Monitoring timeout"

Transfer is still processing. Check later:

```bash
curl http://localhost:8080/v1/messages/MESSAGE_ID | jq '.status'
```

### "jq: command not found"

Install jq:

```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# Or remove jq from scripts
curl ... | python -m json.tool
```

## Advanced Usage

### Batch Transfers

```bash
# Transfer multiple amounts
for amount in 1000000000000000000 2000000000000000000 3000000000000000000; do
  AMOUNT=$amount ./polygon-to-avalanche.sh
  sleep 30
done
```

### Different Chains

```bash
# Create your own route
SOURCE_CHAIN="avalanche-fuji" \
DEST_CHAIN="bnb-testnet" \
./polygon-to-avalanche.sh
```

### Custom API URL

```bash
API_URL="https://bridge.example.com" ./polygon-to-avalanche.sh
```

## Next Steps

1. **Deploy contracts**: See `docs/guides/EVM_DEPLOYMENT.md`
2. **Get testnet tokens**: See `docs/guides/TESTNET_FAUCETS.md`
3. **Monitor with Grafana**: http://localhost:3000
4. **Check transaction history**: `curl http://localhost:8080/v1/messages`

## Support

- **Logs**: Check `logs/api.log` and `logs/relayer.log`
- **Status**: http://localhost:8080/v1/status
- **Docs**: See main documentation in `docs/`
