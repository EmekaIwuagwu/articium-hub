#!/bin/bash

#######################################################################
# Example: Transfer tokens from BNB Testnet to Ethereum Sepolia
#######################################################################

# Configuration
API_URL="http://localhost:8080"
SOURCE_CHAIN="bnb-testnet"
DEST_CHAIN="ethereum-sepolia"
TOKEN_ADDRESS="0xae13d989daC2f0dEbFf460aC112a837C89BAa7cd"  # WBNB
AMOUNT="100000000000000000"  # 0.1 token (18 decimals)
SENDER="0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
RECIPIENT="0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo "========================================"
echo "  BNB -> Ethereum Transfer"
echo "========================================"
echo ""

# Check API health
echo -e "${BLUE}Checking bridge health...${NC}"
HEALTH=$(curl -s "$API_URL/health")
if [[ $HEALTH == *"healthy"* ]]; then
  echo -e "${GREEN}✅ Bridge is healthy${NC}"
else
  echo -e "${YELLOW}⚠️  Bridge health check failed${NC}"
  exit 1
fi

# Initiate transfer
echo ""
echo -e "${BLUE}Initiating transfer...${NC}"
echo "  From: $SOURCE_CHAIN"
echo "  To: $DEST_CHAIN"
echo "  Amount: $AMOUNT (0.1 token)"
echo ""

RESPONSE=$(curl -s -X POST "$API_URL/v1/bridge/token" \
  -H "Content-Type: application/json" \
  -d "{
    \"source_chain\": \"$SOURCE_CHAIN\",
    \"destination_chain\": \"$DEST_CHAIN\",
    \"token_address\": \"$TOKEN_ADDRESS\",
    \"amount\": \"$AMOUNT\",
    \"sender\": \"$SENDER\",
    \"recipient\": \"$RECIPIENT\"
  }")

MESSAGE_ID=$(echo $RESPONSE | jq -r '.message_id')

if [ "$MESSAGE_ID" == "null" ] || [ -z "$MESSAGE_ID" ]; then
  echo -e "${YELLOW}❌ Transfer failed${NC}"
  echo "$RESPONSE" | jq '.'
  exit 1
fi

echo -e "${GREEN}✅ Transfer initiated!${NC}"
echo "  Message ID: $MESSAGE_ID"
echo ""

# Monitor with progress bar
echo -e "${BLUE}Monitoring transfer...${NC}"
for i in {1..60}; do
  sleep 5
  STATUS=$(curl -s "$API_URL/v1/messages/$MESSAGE_ID" | jq -r '.status')

  if [ "$STATUS" == "completed" ]; then
    echo -e "\n${GREEN}🎉 Transfer completed!${NC}"
    curl -s "$API_URL/v1/messages/$MESSAGE_ID" | jq '.'
    exit 0
  elif [ "$STATUS" == "failed" ]; then
    echo -e "\n${YELLOW}❌ Transfer failed${NC}"
    curl -s "$API_URL/v1/messages/$MESSAGE_ID" | jq '.'
    exit 1
  fi

  # Progress bar
  PERCENT=$((i * 100 / 60))
  FILLED=$((i / 3))
  printf "\r  Progress: ["
  for ((j=0; j<20; j++)); do
    if [ $j -lt $FILLED ]; then
      printf "="
    else
      printf " "
    fi
  done
  printf "] %d%% (%s)" $PERCENT "$STATUS"
done

echo ""
echo -e "${YELLOW}⚠️  Monitoring timeout${NC}"
