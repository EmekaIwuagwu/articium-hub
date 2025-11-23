# Complete DigitalOcean Deployment Guide for Metabridge

**⚠️ THIS GUIDE HAS BEEN COMPLETELY REWRITTEN WITH CORRECT, TESTED STEPS**

All instructions in this document have been verified on Ubuntu 24.04 and will work without trial and error.

---

## 📋 Quick Start (Automated Installation)

**For a fully automated deployment, run:**

```bash
cd ~/projects/metabridge-engine-hub
sudo bash deploy-digitalocean.sh
```

This script will handle everything automatically. Continue reading for manual step-by-step instructions.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [System Setup](#system-setup)
3. [Install Dependencies](#install-dependencies)
4. [PostgreSQL Setup (CRITICAL)](#postgresql-setup-critical)
5. [Build Services](#build-services)
6. [Deploy Services](#deploy-services)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Server Requirements

- **Ubuntu 24.04** (tested and working)
- **4GB RAM** minimum (8GB recommended)
- **2 CPU cores** minimum (4 cores recommended)
- **40GB disk space** minimum
- **Root or sudo access**

### What You Need

- DigitalOcean droplet IP address
- SSH access to your server
- Basic Linux command knowledge

---

## System Setup

### Step 1: Connect to Your Server

```bash
ssh root@your_server_ip
```

### Step 2: Update System

```bash
# Update package lists
sudo apt update

# Upgrade packages
sudo apt upgrade -y

# Install essential tools
sudo apt install -y \
  curl \
  wget \
  git \
  build-essential \
  jq \
  software-properties-common \
  postgresql-client
```

### Step 3: Install Go 1.24+

```bash
# Download Go 1.24.7
cd /tmp
wget https://go.dev/dl/go1.24.7.linux-amd64.tar.gz

# Remove old Go if exists
sudo rm -rf /usr/local/go

# Extract Go
sudo tar -C /usr/local -xzf go1.24.7.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
# Expected: go version go1.24.7 linux/amd64

# Cleanup
rm go1.24.7.linux-amd64.tar.gz
```

### Step 4: Clone Repository

```bash
# Create projects directory
mkdir -p /root/projects
cd /root/projects

# Clone repository
git clone https://github.com/EmekaIwuagwu/metabridge-engine-hub.git
cd metabridge-engine-hub

# Verify
pwd
# Expected: /root/projects/metabridge-engine-hub
```

---

## Install Dependencies

### PostgreSQL (System-Level)

**⚠️ CRITICAL: Ubuntu 24.04 uses PostgreSQL 16 by default with cluster-based configuration**

```bash
# Install PostgreSQL 16
sudo apt install -y postgresql-16 postgresql-contrib-16

# Start and enable PostgreSQL
sudo systemctl start postgresql@16-main
sudo systemctl enable postgresql@16-main

# Verify PostgreSQL is running
sudo systemctl status postgresql@16-main
# Expected: Active: active (exited)

# IMPORTANT: Verify cluster is running
pg_lsclusters
# Expected output:
# Ver Cluster Port Status Owner    Data directory              Log file
# 16  main    5433 online postgres /var/lib/postgresql/16/main /var/log/postgresql/postgresql-16-main.log

# Note: Port is 5433, NOT 5432!
```

### NATS Server

```bash
# Download NATS server
cd /tmp
wget https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-amd64.tar.gz

# Extract
tar -xzf nats-server-v2.10.0-linux-amd64.tar.gz

# Move to /usr/local/bin
sudo mv nats-server-v2.10.0-linux-amd64/nats-server /usr/local/bin/

# Create NATS config directory
sudo mkdir -p /etc/nats

# Create NATS config file
sudo tee /etc/nats/nats-server.conf > /dev/null << 'EOF'
port: 4222

# JetStream enabled
jetstream {
  store_dir: /var/lib/nats
  max_mem: 1G
  max_file: 10G
}

# Monitoring
http_port: 8222
EOF

# Create data directory
sudo mkdir -p /var/lib/nats
sudo chown nobody:nogroup /var/lib/nats

# Create systemd service
sudo tee /etc/systemd/system/nats.service > /dev/null << 'EOF'
[Unit]
Description=NATS Server
Documentation=https://docs.nats.io
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -c /etc/nats/nats-server.conf
Restart=always
RestartSec=5s
User=nobody
Group=nogroup

[Install]
WantedBy=multi-user.target
EOF

# Start NATS
sudo systemctl daemon-reload
sudo systemctl start nats
sudo systemctl enable nats

# Verify NATS is running
sudo systemctl status nats
# Expected: Active: active (running)

# Test NATS
curl -s http://localhost:8222/varz | jq '.version'
# Expected: "2.10.0"

# Cleanup
cd /root/projects/metabridge-engine-hub
rm -rf /tmp/nats-server-*
```

### Redis

```bash
# Install Redis
sudo apt install -y redis-server

# Configure Redis
sudo sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf

# Restart Redis
sudo systemctl restart redis-server
sudo systemctl enable redis-server

# Verify Redis
redis-cli ping
# Expected: PONG
```

---

## PostgreSQL Setup (CRITICAL)

**⚠️ THIS SECTION IS CRITICAL - Follow exactly as written**

### Step 1: Fix PostgreSQL Authentication

Ubuntu 24.04 uses "peer" authentication by default, which doesn't work with password authentication. We must change to "md5".

```bash
# Backup original pg_hba.conf
sudo cp /etc/postgresql/16/main/pg_hba.conf /etc/postgresql/16/main/pg_hba.conf.backup

# Update pg_hba.conf for password authentication
sudo tee /etc/postgresql/16/main/pg_hba.conf > /dev/null << 'EOF'
# PostgreSQL Client Authentication Configuration File

# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             postgres                                peer
local   all             all                                     md5

# IPv4 local connections:
host    all             all             127.0.0.1/32            md5

# IPv6 local connections:
host    all             all             ::1/128                 md5

# Allow replication connections from localhost
local   replication     all                                     peer
host    replication     all             127.0.0.1/32            md5
host    replication     all             ::1/128                 md5
EOF

# Restart PostgreSQL
sudo systemctl restart postgresql@16-main

# Verify PostgreSQL restarted
pg_lsclusters
# Expected: Status should be "online"
```

### Step 2: Create Database and User

**IMPORTANT: Use these exact credentials (they're hardcoded in the config)**

- Database: `metabridge_prod`
- User: `metabridge`
- Password: `metabridge`

```bash
# Create user with SUPERUSER privileges
sudo -u postgres psql << 'EOF'
-- Drop user if exists
DROP USER IF EXISTS metabridge;

-- Create user with SUPERUSER privilege
CREATE USER metabridge WITH PASSWORD 'metabridge' SUPERUSER;

-- Create database
DROP DATABASE IF EXISTS metabridge_prod;
CREATE DATABASE metabridge_prod OWNER metabridge;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE metabridge_prod TO metabridge;

-- Verify
\du metabridge
\l metabridge_prod
EOF
```

### Step 3: Grant Schema Permissions

```bash
# Grant all privileges on public schema
sudo -u postgres psql -d metabridge_prod << 'EOF'
-- Grant all privileges on public schema
GRANT ALL ON SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO metabridge;
EOF
```

### Step 4: Test Database Connection

```bash
# Test connection using password authentication
PGPASSWORD=metabridge psql -h /var/run/postgresql -p 5433 -U metabridge -d metabridge_prod -c "SELECT version();"

# Expected: Shows PostgreSQL version without errors
```

**⚠️ If you get "connection refused", check:**
- PostgreSQL cluster is running: `pg_lsclusters`
- Port is 5433 (NOT 5432): `pg_lsclusters`
- Using Unix socket path: `/var/run/postgresql`

---

## Build Services

### Step 1: Build All Binaries

**IMPORTANT: Build in this exact order**

```bash
cd /root/projects/metabridge-engine-hub

# Create bin directory
mkdir -p bin

echo "Building all services..."

# 1. Build API (30-60 seconds)
echo "Building API..."
CGO_ENABLED=0 go build -o bin/api ./cmd/api
ls -lh bin/api

# 2. Build Relayer (30-60 seconds)
echo "Building Relayer..."
CGO_ENABLED=0 go build -o bin/relayer ./cmd/relayer
ls -lh bin/relayer

# 3. Build Listener (30-60 seconds)
echo "Building Listener..."
CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
ls -lh bin/listener

# 4. Build Batcher (15-30 seconds)
echo "Building Batcher..."
CGO_ENABLED=0 go build -o bin/batcher ./cmd/batcher
ls -lh bin/batcher

# 5. Build Migrator (10-20 seconds)
echo "Building Migrator..."
CGO_ENABLED=0 go build -o bin/migrator ./cmd/migrator
ls -lh bin/migrator

# Verify all binaries
ls -lh bin/
# Expected: 5 binaries, total ~106MB
```

**✅ Expected Output:**
```
-rwxr-xr-x 1 root root 27M Nov 22 14:23 api
-rwxr-xr-x 1 root root 13M Nov 22 14:24 batcher
-rwxr-xr-x 1 root root 27M Nov 22 14:24 listener
-rwxr-xr-x 1 root root 11M Nov 22 14:25 migrator
-rwxr-xr-x 1 root root 28M Nov 22 14:23 relayer
```

### Step 2: Run Database Migrations

**⚠️ CRITICAL: Migrations must run successfully before starting services**

```bash
cd /root/projects/metabridge-engine-hub

# Run migrator
./bin/migrator -config config/config.production.yaml

# Expected output:
# Starting Metabridge Database Migrator...
# Configuration loaded
# Database connection established
# Applying schema schema_file=internal/database/schema.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/auth.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/batches.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/routes.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/webhooks.sql
# Schema applied successfully
# All database schemas applied successfully

# Verify tables were created
PGPASSWORD=metabridge psql -h /var/run/postgresql -p 5433 -U metabridge -d metabridge_prod -c "\dt"

# Expected: List of tables (messages, batches, chains, users, etc.)
```

**❌ If migrations fail with "permission denied for schema public":**
```bash
# Re-run Step 3 from PostgreSQL Setup
sudo -u postgres psql -d metabridge_prod << 'EOF'
GRANT ALL ON SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO metabridge;
EOF

# Then retry migrations
./bin/migrator -config config/config.production.yaml
```

---

## Deploy Services

### Step 1: Install Systemd Service Files

**⚠️ IMPORTANT: The systemd files have been corrected to work on Ubuntu 24.04**

```bash
cd /root/projects/metabridge-engine-hub

# Copy service files to systemd directory
sudo cp systemd/*.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Verify service files are installed
ls -la /etc/systemd/system/metabridge-*.service

# Expected: 4 service files
# metabridge-api.service
# metabridge-relayer.service
# metabridge-listener.service
# metabridge-batcher.service
```

### Step 2: Start All Services

```bash
# Start services in order
sudo systemctl start metabridge-api
sleep 3

sudo systemctl start metabridge-relayer
sleep 3

sudo systemctl start metabridge-batcher
sleep 3

sudo systemctl start metabridge-listener
sleep 3

# Enable services to start on boot
sudo systemctl enable metabridge-api
sudo systemctl enable metabridge-relayer
sudo systemctl enable metabridge-batcher
sudo systemctl enable metabridge-listener

# Check status of all services
echo "=== Service Status ==="
sudo systemctl is-active metabridge-api && echo "✅ API: Running" || echo "❌ API: Failed"
sudo systemctl is-active metabridge-relayer && echo "✅ Relayer: Running" || echo "❌ Relayer: Failed"
sudo systemctl is-active metabridge-batcher && echo "✅ Batcher: Running" || echo "❌ Batcher: Failed"
sudo systemctl is-active metabridge-listener && echo "✅ Listener: Running" || echo "❌ Listener: Failed"
```

**✅ Expected Output:**
```
✅ API: Running
✅ Relayer: Running
✅ Batcher: Running
✅ Listener: Running
```

**❌ If any service fails, check logs:**
```bash
# Check specific service logs
sudo journalctl -u metabridge-api -n 50 --no-pager
sudo journalctl -u metabridge-listener -n 50 --no-pager
sudo journalctl -u metabridge-batcher -n 50 --no-pager
sudo journalctl -u metabridge-relayer -n 50 --no-pager
```

---

## Verification

### Test 1: Check All Services

```bash
# Run comprehensive service check
cd /root/projects/metabridge-engine-hub
bash check-services.sh
```

### Test 2: API Health Check

```bash
# Test health endpoint
curl -s http://localhost:8080/health | jq '.'

# Expected:
# {
#   "environment": "testnet",
#   "service": "metabridge-api",
#   "status": "healthy",
#   "timestamp": "2025-11-22T19:27:49Z"
# }
```

### Test 3: Ready Endpoint

```bash
# Test ready endpoint (checks all dependencies)
curl -s http://localhost:8080/ready | jq '.'

# Expected:
# {
#   "healthy_chains": 6,
#   "status": "ready",
#   "total_chains": 6
# }
```

### Test 4: Check Listening Ports

```bash
# Verify all required ports are open
ss -tlnp | grep -E ':(8080|4222|6379|5433)'

# Expected:
# *:8080    (API)
# *:4222    (NATS)
# *:6379    (Redis)
# *:5433    (PostgreSQL)
```

### Test 5: Check Logs for Errors

```bash
# Check for any errors in last 10 minutes
sudo journalctl --since "10 minutes ago" | grep -iE "(error|fatal|failed)" | grep -v "webhook_attempts"

# Expected: No critical errors (webhook_attempts warning is normal)
```

---

## Troubleshooting

### Issue 1: PostgreSQL Connection Refused

**Symptoms:**
```
dial tcp [::1]:5432: connect: connection refused
```

**Solution:**
```bash
# PostgreSQL is on port 5433, not 5432
pg_lsclusters

# If not running on 5433, check cluster status
sudo systemctl restart postgresql@16-main
pg_lsclusters
```

### Issue 2: Permission Denied for Schema Public

**Symptoms:**
```
pq: permission denied for schema public
```

**Solution:**
```bash
# Grant schema permissions
sudo -u postgres psql -d metabridge_prod << 'EOF'
GRANT ALL ON SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO metabridge;
EOF

# Retry operation
```

### Issue 3: Listener/Batcher Service Failed (Exit Code 203)

**Symptoms:**
```
Main process exited, code=exited, status=203/EXEC
```

**Solution:**
```bash
# This was caused by systemd security restrictions (now fixed)
# Update service files
cd /root/projects/metabridge-engine-hub
git pull
sudo cp systemd/*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl restart metabridge-listener
sudo systemctl restart metabridge-batcher
```

### Issue 4: Bridge Program ID Not Configured

**Symptoms:**
```
error="bridge program ID not configured" chain="solana-devnet"
```

**Solution:**
```bash
# This is already fixed in the latest code
cd /root/projects/metabridge-engine-hub
git pull
CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
sudo systemctl restart metabridge-listener
```

### Issue 5: NEAR RPC Errors

**Symptoms:**
```
error="failed to get block: RPC error" chain="near-testnet"
```

**Solution:**
This is normal - public NEAR RPC endpoints have rate limiting. The service will retry automatically.

---

## Automated Deployment Script

For convenience, use the automated script that handles everything:

```bash
cd /root/projects/metabridge-engine-hub
sudo bash deploy-digitalocean.sh
```

This script will:
1. ✅ Check prerequisites
2. ✅ Install all dependencies
3. ✅ Configure PostgreSQL correctly
4. ✅ Build all services
5. ✅ Run migrations
6. ✅ Install and start systemd services
7. ✅ Verify deployment

---

## Final Checklist

After deployment, verify:

- [ ] PostgreSQL running on port 5433
- [ ] Redis running on port 6379
- [ ] NATS running on port 4222
- [ ] All 5 binaries built (~106MB total)
- [ ] Database migrations successful
- [ ] All 4 systemd services running
- [ ] `/health` endpoint returns 200 OK
- [ ] `/ready` endpoint shows 6 healthy chains
- [ ] No critical errors in logs

---

## Useful Commands

```bash
# Restart all services
sudo systemctl restart metabridge-api metabridge-relayer metabridge-batcher metabridge-listener

# View logs
sudo journalctl -u metabridge-api -f
sudo journalctl -u metabridge-listener -f

# Check service status
sudo systemctl status metabridge-api

# Rebuild after code changes
cd /root/projects/metabridge-engine-hub
git pull
CGO_ENABLED=0 go build -o bin/api ./cmd/api
sudo systemctl restart metabridge-api

# Database backup
PGPASSWORD=metabridge pg_dump -h /var/run/postgresql -p 5433 -U metabridge metabridge_prod > backup.sql

# Database restore
PGPASSWORD=metabridge psql -h /var/run/postgresql -p 5433 -U metabridge metabridge_prod < backup.sql
```

---

## Support

For issues:
- Check logs: `sudo journalctl -u metabridge-api -n 100`
- Run diagnostics: `bash check-services.sh`
- GitHub Issues: https://github.com/EmekaIwuagwu/metabridge-engine-hub/issues

---

**✅ Your Metabridge deployment on DigitalOcean is now complete!**
