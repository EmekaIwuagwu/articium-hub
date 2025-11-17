#!/bin/bash

#######################################################################
# Example: Transfer tokens from Polygon Amoy to Avalanche Fuji
#######################################################################

# Configuration
API_URL="http://localhost:8080"
SOURCE_CHAIN="polygon-amoy"
DEST_CHAIN="avalanche-fuji"
TOKEN_ADDRESS="0x0000000000000000000000000000000000001010"  # WMATIC
AMOUNT="1000000000000000000"  # 1 token (18 decimals)
SENDER="0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
RECIPIENT="0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo ""
echo "========================================"
echo "  Polygon -> Avalanche Transfer"
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
echo "  Amount: $AMOUNT (1 token)"
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

# Extract message ID
MESSAGE_ID=$(echo $RESPONSE | jq -r '.message_id')

if [ "$MESSAGE_ID" == "null" ] || [ -z "$MESSAGE_ID" ]; then
  echo -e "${YELLOW}❌ Transfer failed${NC}"
  echo "$RESPONSE" | jq '.'
  exit 1
fi

echo -e "${GREEN}✅ Transfer initiated!${NC}"
echo "  Message ID: $MESSAGE_ID"
echo ""

# Monitor status
echo -e "${BLUE}Monitoring transfer...${NC}"
echo "(This may take several minutes)"
echo ""

for i in {1..60}; do
  sleep 5

  STATUS=$(curl -s "$API_URL/v1/messages/$MESSAGE_ID" | jq -r '.status')

  if [ "$STATUS" == "completed" ]; then
    echo -e "${GREEN}🎉 Transfer completed!${NC}"
    curl -s "$API_URL/v1/messages/$MESSAGE_ID" | jq '.'
    exit 0
  elif [ "$STATUS" == "failed" ]; then
    echo -e "${YELLOW}❌ Transfer failed${NC}"
    curl -s "$API_URL/v1/messages/$MESSAGE_ID" | jq '.'
    exit 1
  else
    echo "  Status: $STATUS ($i/60)"
  fi
done

echo ""
echo -e "${YELLOW}⚠️  Monitoring timeout${NC}"
echo "Check status manually:"
echo "  curl $API_URL/v1/messages/$MESSAGE_ID | jq '.'"
echo ""
