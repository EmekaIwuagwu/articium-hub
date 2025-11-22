#!/bin/bash

# Quick setup script for Metabridge - Run this first!

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║           Metabridge Quick Setup                           ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root: sudo bash scripts/quick-setup.sh"
    exit 1
fi

# Step 1: Install dependencies
echo "Step 1/5: Installing dependencies..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
bash scripts/install-dependencies.sh
echo ""

# Step 2: Run migrations
echo "Step 2/5: Running database migrations..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ -f "./bin/migrator" ]; then
    ./bin/migrator -config config/config.production.yaml || {
        echo "⚠️  Migrations failed - this is normal if database was already migrated"
    }
else
    echo "⚠️  Migrator binary not found. Run: make build"
fi
echo ""

# Step 3: Install systemd services
echo "Step 3/5: Installing systemd services..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cp systemd/*.service /etc/systemd/system/
systemctl daemon-reload
echo "✓ Service files installed"
echo ""

# Step 4: Enable services
echo "Step 4/5: Enabling services for auto-start..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
systemctl enable metabridge-api
systemctl enable metabridge-relayer
systemctl enable metabridge-listener
systemctl enable metabridge-batcher
echo "✓ Services enabled"
echo ""

# Step 5: Start services
echo "Step 5/5: Starting services..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
systemctl start metabridge-api
sleep 3
systemctl start metabridge-relayer
systemctl start metabridge-listener
systemctl start metabridge-batcher
echo "✓ Services started"
echo ""

# Check status
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Service Status:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
systemctl is-active metabridge-api && echo "  ✓ API:      running" || echo "  ✗ API:      failed"
systemctl is-active metabridge-relayer && echo "  ✓ Relayer:  running" || echo "  ✗ Relayer:  failed"
systemctl is-active metabridge-listener && echo "  ✓ Listener: running" || echo "  ✗ Listener: failed"
systemctl is-active metabridge-batcher && echo "  ✓ Batcher:  running" || echo "  ✗ Batcher:  failed"
echo ""

# Test API
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Testing API endpoint..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sleep 2
if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
    echo "✅ API is responding!"
    echo ""
    echo "API Health:"
    curl -s http://localhost:8080/api/v1/health | jq . 2>/dev/null || curl -s http://localhost:8080/api/v1/health
else
    echo "⚠️  API is not responding yet. Check logs:"
    echo "   sudo journalctl -u metabridge-api -n 50"
fi
echo ""

echo "╔════════════════════════════════════════════════════════════╗"
echo "║                 ✅ SETUP COMPLETE!                         ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Useful commands:"
echo "  Check status:  systemctl status metabridge-api"
echo "  View logs:     sudo journalctl -u metabridge-api -f"
echo "  Restart:       sudo systemctl restart metabridge-api"
echo "  Check health:  curl http://localhost:8080/api/v1/health"
echo ""
