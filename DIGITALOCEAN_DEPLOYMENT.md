# Complete DigitalOcean Deployment Guide for Metabridge

**Your Droplet IP**: `159.65.73.133`

This guide will take you from SSH login to a fully running bridge in ~30 minutes.

---

## 📋 Quick Reference

### What You'll Build
- 5 Go binaries (API, Relayer, Listener, Batcher, Migrator)
- Total binary size: ~106 MB
- Expected build time: 2-5 minutes

### Key Expected Responses

**✅ Successful Compilation:**
- No error messages
- Commands complete silently (silence = success!)
- Binary files created in `bin/` directory

**✅ Successful Service Start:**
- `systemctl status` shows "active (running)"
- Health endpoint returns `{"status":"healthy"}`
- All 6 chains show `"healthy": true`

**✅ Successful Database Setup:**
- Tables created: messages, batches, users, api_keys, routes, webhooks
- Admin user exists
- Database size: ~50-100 MB fresh install

### Quick Health Check Commands

```bash
# Check all services at once
sudo systemctl status metabridge-api metabridge-relayer | grep "Active:"
# Expected: Active: active (running) for both

# Check API is responding
curl -s http://159.65.73.133:8080/health | grep status
# Expected: "status":"healthy"

# Check all chains
curl -s http://159.65.73.133:8080/v1/chains/status | jq 'to_entries[] | {chain: .key, healthy: .value.healthy}'
# Expected: All show "healthy": true
```

### Common Expected Outputs Reference

| Command | Expected Output | Meaning |
|---------|----------------|---------|
| `go build ...` | (silence) | ✅ Compilation successful |
| `systemctl status` | `Active: active (running)` | ✅ Service running |
| `curl /health` | `{"status":"healthy"}` | ✅ API responding |
| `docker ps` | `Up (healthy)` | ✅ Container running |
| `psql -c "\dt"` | List of tables | ✅ Database initialized |

### Detailed Documentation References

For comprehensive compilation information, troubleshooting, and expected responses, see:
- **Step 13**: Detailed compilation process and expected build outputs
- **Step 16**: Comprehensive testing with all expected responses
- `Documentations/COMPILATION_TEST_REPORT.md`: Full compilation report
- `Documentations/BUILD_VERIFICATION.md`: Build verification checklist

---

## Step 1: Connect to Your Droplet

```bash
# SSH into your DigitalOcean droplet
ssh root@159.65.73.133

# If you're using a non-root user:
# ssh your-username@159.65.73.133
```

If prompted about host authenticity, type `yes` and press Enter.

## Step 2: System Update & Upgrade

```bash
# Update package lists
sudo apt update

# Upgrade all packages (this may take 5-10 minutes)
sudo apt upgrade -y

# Install essential tools
sudo apt install -y \
  curl \
  wget \
  git \
  build-essential \
  jq \
  unzip \
  software-properties-common \
  apt-transport-https \
  ca-certificates \
  gnupg \
  lsb-release \
  htop \
  vim

# Set timezone to UTC
sudo timedatectl set-timezone UTC

# Verify
date
```

## Step 3: Install Go 1.21+

```bash
# Download Go
cd ~
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz

# Remove old Go installation (if exists)
sudo rm -rf /usr/local/go

# Extract Go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz

# Add Go to PATH
cat >> ~/.bashrc << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
EOF

# Apply changes
source ~/.bashrc

# Verify Go installation
go version
# Should output: go version go1.21.6 linux/amd64

# Clean up
rm go1.21.6.linux-amd64.tar.gz
```

## Step 4: Install Node.js 18+

```bash
# Install Node.js via NodeSource
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Verify installation
node --version  # Should be v18.x.x
npm --version   # Should be 9.x.x or higher

# Install Yarn globally (optional)
sudo npm install -g yarn
```

## Step 5: Install Docker & Docker Compose

```bash
# Add Docker's official GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add Docker repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Start Docker
sudo systemctl start docker
sudo systemctl enable docker

# Add current user to docker group (if not root)
sudo usermod -aG docker $USER

# Verify installation
docker --version
docker compose version

# Test Docker
sudo docker run hello-world
```

## Step 6: Clone Metabridge Repository

```bash
# Create project directory
mkdir -p ~/projects
cd ~/projects

# Clone repository
git clone https://github.com/EmekaIwuagwu/metabridge-engine-hub.git
cd metabridge-engine-hub

# Check repository
ls -la
git status
git branch

# Check what branch you're on and switch to main if needed
git checkout main
```

## Step 7: Install Project Dependencies

```bash
# Ensure you're in the project root
cd ~/projects/metabridge-engine-hub

# Install Go dependencies
go mod download
go mod verify

# Install smart contract dependencies (EVM)
cd contracts/evm
npm install
cd ../..

# Verify installation
echo "✅ Dependencies installed successfully"
```

## Step 8: Configure Environment Variables

```bash
cd ~/projects/metabridge-engine-hub

# Copy environment template
cp .env.example .env.production

# Edit environment file
nano .env.production
```

**Paste this configuration** (customize the marked fields):

```bash
# Environment
BRIDGE_ENVIRONMENT=production

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=bridge_user
DB_PASSWORD=YourStrongPassword123!  # ⚠️ CHANGE THIS
DB_NAME=metabridge_production
DB_SSLMODE=disable

# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# JWT Authentication (generate with: openssl rand -hex 32)
JWT_SECRET=your_super_secret_jwt_key_at_least_32_characters_long_here  # ⚠️ CHANGE THIS
JWT_EXPIRATION_HOURS=24

# CORS (allow all for testing, restrict in production)
CORS_ALLOWED_ORIGINS=*

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100
REQUIRE_AUTH=false  # Set to true after creating admin user
API_KEY_ENABLED=true

# RPC Endpoints - Get free API keys from these services
# Alchemy: https://www.alchemy.com/
# Infura: https://infura.io/
# Helius: https://helius.dev/
ALCHEMY_API_KEY=your_alchemy_api_key_here  # ⚠️ GET FREE KEY
INFURA_API_KEY=your_infura_api_key_here    # ⚠️ GET FREE KEY
HELIUS_API_KEY=your_helius_api_key_here    # ⚠️ GET FREE KEY (optional)

# Chain RPC URLs (Testnet)
POLYGON_RPC_URL=https://rpc-amoy.polygon.technology/
BNB_RPC_URL=https://data-seed-prebsc-1-s1.binance.org:8545/
AVALANCHE_RPC_URL=https://api.avax-test.network/ext/bc/C/rpc
ETHEREUM_RPC_URL=https://sepolia.infura.io/v3/${INFURA_API_KEY}
SOLANA_RPC_URL=https://api.devnet.solana.com
NEAR_RPC_URL=https://rpc.testnet.near.org

# Smart Contract Addresses (leave empty for now)
POLYGON_BRIDGE_CONTRACT=
BNB_BRIDGE_CONTRACT=
AVALANCHE_BRIDGE_CONTRACT=
ETHEREUM_BRIDGE_CONTRACT=
SOLANA_BRIDGE_PROGRAM=
NEAR_BRIDGE_CONTRACT=

# Validator Configuration (generate a new test wallet)
VALIDATOR_PRIVATE_KEY=your_private_key_here  # ⚠️ GENERATE NEW TEST WALLET

# NATS Configuration
NATS_URL=nats://localhost:4222

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

**Save and exit**: Press `Ctrl+X`, then `Y`, then `Enter`

### Generate JWT Secret

```bash
# Generate a secure JWT secret
openssl rand -hex 32

# Copy the output and paste it into your .env.production file as JWT_SECRET
```

## Step 9: Create Docker Compose File

```bash
cd ~/projects/metabridge-engine-hub

# Create docker-compose.production.yaml
cat > docker-compose.production.yaml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15
    container_name: metabridge-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres_admin_password
      POSTGRES_DB: metabridge_production
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  nats:
    image: nats:2.10
    container_name: metabridge-nats
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["-js", "-m", "8222"]
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: metabridge-redis
    ports:
      - "6379:6379"
    restart: always
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
EOF

echo "✅ Docker Compose file created"
```

## Step 10: Start Infrastructure Services

```bash
cd ~/projects/metabridge-engine-hub

# Start PostgreSQL, NATS, and Redis
sudo docker compose -f docker-compose.production.yaml up -d

# Wait for services to start (30 seconds)
echo "⏳ Waiting for services to start..."
sleep 30

# Check service status
sudo docker compose -f docker-compose.production.yaml ps

# You should see all three services running (Up)

# Check logs if needed
sudo docker compose -f docker-compose.production.yaml logs
```

## Step 11: Initialize Database

```bash
cd ~/projects/metabridge-engine-hub

# Create database and user
sudo docker exec -i metabridge-postgres psql -U postgres << EOF
CREATE DATABASE metabridge_production;
CREATE USER bridge_user WITH ENCRYPTED PASSWORD 'YourStrongPassword123!';
GRANT ALL PRIVILEGES ON DATABASE metabridge_production TO bridge_user;
ALTER DATABASE metabridge_production OWNER TO bridge_user;
\c metabridge_production
GRANT ALL ON SCHEMA public TO bridge_user;
EOF

# Run main database schema
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/schema.sql

# Run authentication schema
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/auth.sql

# Verify tables were created
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "\dt"

# You should see tables like: messages, batches, webhooks, routes, users, api_keys, etc.
```

## Step 12: Create Admin User

```bash
# Install bcrypt tool for password hashing
go install github.com/bitnami/bcrypt-cli@latest

# Hash your admin password (replace 'admin123' with your desired password)
~/go/bin/bcrypt-cli admin123

# Copy the hash output (starts with $2a$...)
# Example output: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Insert admin user (replace <YOUR_BCRYPT_HASH> with the hash from above)
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
VALUES (
  'admin-001',
  'admin@metabridge.local',
  'System Administrator',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
  'admin',
  true,
  NOW(),
  NOW()
);
EOF

# Verify user was created
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT id, email, role FROM users;"
```

## Step 13: Build Bridge Services

This is a critical step where you compile all the Go binaries for your bridge system.

### Build Process Overview

The Metabridge Engine consists of 5 main binaries:
1. **metabridge-api** - Main API server (handles HTTP requests)
2. **metabridge-relayer** - Message relayer (processes cross-chain messages)
3. **metabridge-listener** - Blockchain listener (monitors chain events)
4. **metabridge-batcher** - Batch aggregator (optimizes gas costs)
5. **metabridge-migrator** - Database migrator (sets up database schema)

### Expected Build Time
- **Total Time**: 2-5 minutes (depending on your server specs)
- **Per Binary**: 30-60 seconds each
- **Download Time**: Additional 1-2 minutes for first build (dependencies)

### Build Commands

```bash
cd ~/projects/metabridge-engine-hub

# Create bin directory
mkdir -p bin

# Build 1: API Server
echo "🔨 Building API server..."
CGO_ENABLED=0 go build -o bin/metabridge-api cmd/api/main.go

# Expected Output:
# (Downloading dependencies on first build - you'll see progress bars)
# go: downloading github.com/ethereum/go-ethereum v1.13.8
# go: downloading github.com/gorilla/mux v1.8.1
# go: downloading github.com/rs/zerolog v1.31.0
# ... (20-30 more packages)
# (Then silence as it compiles - this is normal!)
# (After 30-60 seconds, command completes with no output = SUCCESS)

echo "✅ API server built"

# Build 2: Relayer Service
echo "🔨 Building relayer..."
CGO_ENABLED=0 go build -o bin/metabridge-relayer cmd/relayer/main.go

# Expected Output:
# (Faster this time since dependencies are cached)
# (15-30 seconds of silence)
# (Completes with no output = SUCCESS)

echo "✅ Relayer built"

# Build 3: Listener Service
echo "🔨 Building listener..."
CGO_ENABLED=0 go build -o bin/metabridge-listener cmd/listener/main.go

# Expected Output:
# (15-30 seconds of compilation)
# (No output = SUCCESS)

echo "✅ Listener built"

# Build 4: Batcher Service
echo "🔨 Building batcher..."
CGO_ENABLED=0 go build -o bin/metabridge-batcher cmd/batcher/main.go

# Expected Output:
# (10-20 seconds of compilation)
# (No output = SUCCESS)

echo "✅ Batcher built"

# Build 5: Database Migrator
echo "🔨 Building migrator..."
CGO_ENABLED=0 go build -o bin/metabridge-migrator cmd/migrator/main.go

# Expected Output:
# (10-20 seconds of compilation)
# (No output = SUCCESS)

echo "✅ Migrator built"

echo ""
echo "==================== BUILD VERIFICATION ===================="
echo ""

# Verify all binaries were created
ls -lh bin/

# Expected Output:
# total 106M
# -rwxr-xr-x 1 root root 27M Nov 22 14:23 metabridge-api
# -rwxr-xr-x 1 root root 13M Nov 22 14:24 metabridge-batcher
# -rwxr-xr-x 1 root root 27M Nov 22 14:24 metabridge-listener
# -rwxr-xr-x 1 root root 11M Nov 22 14:25 metabridge-migrator
# -rwxr-xr-x 1 root root 28M Nov 22 14:23 metabridge-relayer

echo ""
echo "Checking binary types..."
file bin/*

# Expected Output:
# bin/metabridge-api:      ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-batcher:  ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-listener: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-migrator: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-relayer:  ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped

echo ""
echo "Testing binaries respond to --help..."

# Test each binary
bin/metabridge-api --help 2>&1 | head -5
# Expected: Shows help text or "Usage:" message

bin/metabridge-relayer --help 2>&1 | head -5
# Expected: Shows help text or "Usage:" message

echo ""
echo "✅ All binaries built successfully!"
echo ""
echo "Binary Sizes:"
du -h bin/* | column -t
echo ""
echo "Total Size: $(du -sh bin/ | awk '{print $1}')"
echo ""
```

### Alternative: Build All at Once Using Makefile

```bash
# Use the Makefile to build everything
make build

# Expected Output:
# Building Go binaries...
# CGO_ENABLED=0 go build -o bin/api ./cmd/api
# CGO_ENABLED=0 go build -o bin/relayer ./cmd/relayer
# CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
# CGO_ENABLED=0 go build -o bin/batcher ./cmd/batcher
# CGO_ENABLED=0 go build -o bin/migrator ./cmd/migrator
# Build complete! Binaries in ./bin/
# total 106M
# -rwxr-xr-x 1 root root 27M Nov 22 14:23 api
# -rwxr-xr-x 1 root root 13M Nov 22 14:24 batcher
# -rwxr-xr-x 1 root root 27M Nov 22 14:24 listener
# -rwxr-xr-x 1 root root 11M Nov 22 14:25 migrator
# -rwxr-xr-x 1 root root 28M Nov 22 14:23 relayer
```

### What Does "SUCCESS" Look Like?

✅ **Successful Build Indicators:**
- No error messages displayed
- Command completes and returns to shell prompt
- Binary file created in `bin/` directory
- Binary is executable (shown as green in `ls` with colors)
- Binary responds to `--help` flag
- Binary shows "ELF 64-bit LSB executable" in `file` command

❌ **Build Failure Indicators:**
- Error messages containing "undefined:", "not found", "cannot find package"
- No binary file created
- Build process exits early
- Red error text displayed

### Common Build Issues & Solutions

#### Issue 1: Cannot Download Packages

**Error Message:**
```
go: github.com/ethereum/go-ethereum@v1.13.8: Get "https://proxy.golang.org/...": dial tcp: lookup proxy.golang.org: no such host
```

**Solution:**
```bash
# Check DNS settings
cat /etc/resolv.conf

# Fix DNS if needed
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 8.8.4.4" | sudo tee -a /etc/resolv.conf

# Test connectivity
ping -c 3 proxy.golang.org

# Retry build
go clean -modcache
go build -o bin/metabridge-api cmd/api/main.go
```

#### Issue 2: Out of Memory

**Error Message:**
```
signal: killed
```

**Solution:**
```bash
# Check available memory
free -h

# Add swap space if needed (2GB)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Make permanent
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# Retry build
go build -o bin/metabridge-api cmd/api/main.go
```

#### Issue 3: Compilation Errors

**Error Message:**
```
# github.com/EmekaIwuagwu/metabridge-hub/internal/api
internal/api/handlers.go:123:45: undefined: SomeFunction
```

**Solution:**
```bash
# Make sure you're on the correct branch
git status
git branch

# Pull latest code
git pull origin main

# Clean and rebuild
go clean -cache
go mod tidy
go build -o bin/metabridge-api cmd/api/main.go
```

#### Issue 4: Permission Denied

**Error Message:**
```
permission denied
```

**Solution:**
```bash
# Make bin directory writable
chmod 755 bin/
chmod 644 bin/*

# Or run with sudo
sudo go build -o bin/metabridge-api cmd/api/main.go
```

### Build Verification Checklist

After building, verify everything is correct:

```bash
# ✅ 1. Check all 5 binaries exist
ls bin/ | wc -l
# Expected: 5

# ✅ 2. Check total size is reasonable
du -sh bin/
# Expected: 100M-120M (statically linked binaries)

# ✅ 3. Check binaries are executable
ls -la bin/ | grep rwx
# Expected: All 5 files show -rwxr-xr-x

# ✅ 4. Check architecture matches your server
file bin/metabridge-api
# Expected: x86-64 (for most servers)
# If you see "ARM aarch64", that's also fine (for ARM servers)

# ✅ 5. Check binaries are statically linked
ldd bin/metabridge-api
# Expected: "not a dynamic executable" or "statically linked"

# ✅ 6. Test binary execution
bin/metabridge-api --version 2>&1
# Expected: Version info or error message (but binary runs)

# ✅ 7. Check Go build cache
go clean -cache -n
# Shows what would be cleaned (means cache exists)
```

### Performance Metrics

**Expected Build Performance:**

| Binary | Size | Build Time | Dependencies |
|--------|------|------------|--------------|
| metabridge-api | ~27 MB | 30-60s | High (HTTP, DB, chains) |
| metabridge-relayer | ~28 MB | 30-60s | High (all chains, NATS) |
| metabridge-listener | ~27 MB | 30-60s | High (all chains, events) |
| metabridge-batcher | ~13 MB | 15-30s | Medium (batch logic) |
| metabridge-migrator | ~11 MB | 10-20s | Low (DB only) |

**Total:** ~106 MB, 2-5 minutes

### Advanced: Optimized Build

For production deployment with optimizations:

```bash
# Build with optimizations and version info
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse HEAD)

go build \
  -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
  -trimpath \
  -o bin/metabridge-api \
  cmd/api/main.go

# Explanation:
# -ldflags="-s -w"  : Strip debug symbols (smaller binary)
# -X main.Version   : Inject version information
# -trimpath         : Remove file system paths from binary
# Result: Smaller binaries (~20-25% reduction)
```

### Next Steps

Once all binaries are built successfully, proceed to Step 14 to create systemd services that will run these binaries automatically.

## Step 14: Create Systemd Services

### API Server Service

```bash
sudo tee /etc/systemd/system/metabridge-api.service > /dev/null << EOF
[Unit]
Description=Metabridge API Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=$USER
WorkingDirectory=/home/$USER/projects/metabridge-engine-hub
ExecStart=/home/$USER/projects/metabridge-engine-hub/bin/metabridge-api
EnvironmentFile=/home/$USER/projects/metabridge-engine-hub/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

### Relayer Service

```bash
sudo tee /etc/systemd/system/metabridge-relayer.service > /dev/null << EOF
[Unit]
Description=Metabridge Relayer Service
After=network.target docker.service metabridge-api.service
Requires=docker.service

[Service]
Type=simple
User=$USER
WorkingDirectory=/home/$USER/projects/metabridge-engine-hub
ExecStart=/home/$USER/projects/metabridge-engine-hub/bin/metabridge-relayer --config /home/$USER/projects/metabridge-engine-hub/config/config.testnet.yaml
EnvironmentFile=/home/$USER/projects/metabridge-engine-hub/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

### Enable and Start Services

```bash
# Reload systemd daemon
sudo systemctl daemon-reload

# Enable services to start on boot
sudo systemctl enable metabridge-api
sudo systemctl enable metabridge-relayer

# Start services
sudo systemctl start metabridge-api
sudo systemctl start metabridge-relayer

# Check status
sudo systemctl status metabridge-api
sudo systemctl status metabridge-relayer
```

## Step 15: Configure Firewall

```bash
# Install UFW if not already installed
sudo apt install -y ufw

# Allow SSH (CRITICAL - do this first!)
sudo ufw allow 22/tcp

# Allow HTTP
sudo ufw allow 80/tcp

# Allow HTTPS
sudo ufw allow 443/tcp

# Allow API server
sudo ufw allow 8080/tcp

# Enable firewall
sudo ufw --force enable

# Check firewall status
sudo ufw status verbose
```

## Step 16: Run Comprehensive Tests

This section provides detailed tests to verify your deployment is working correctly. Each test includes the exact command, expected output, and what to do if the test fails.

### Test 1: Infrastructure Health Checks

These tests verify that all your infrastructure services (PostgreSQL, NATS, Redis) are running properly.

```bash
echo "========================================="
echo "Test 1: Infrastructure Health Checks"
echo "========================================="
echo ""

# Check all Docker containers are running
echo "Checking Docker containers..."
sudo docker compose -f ~/projects/metabridge-engine-hub/docker-compose.production.yaml ps

# Expected output:
# NAME                    IMAGE              COMMAND                  SERVICE    CREATED          STATUS                    PORTS
# metabridge-nats         nats:2.10          "/nats-server -js -m…"   nats       10 minutes ago   Up 10 minutes (healthy)   0.0.0.0:4222->4222/tcp, 0.0.0.0:8222->8222/tcp
# metabridge-postgres     postgres:15        "docker-entrypoint.s…"   postgres   10 minutes ago   Up 10 minutes (healthy)   0.0.0.0:5432->5432/tcp
# metabridge-redis        redis:7-alpine     "docker-entrypoint.s…"   redis      10 minutes ago   Up 10 minutes (healthy)   0.0.0.0:6379->6379/tcp

# What to look for:
# ✅ STATUS column shows "Up" for all containers
# ✅ "(healthy)" appears next to each container
# ❌ If "Exit" or "Restarting" appears, container has issues

echo ""
echo "Testing PostgreSQL connection..."
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT version();"

# Expected output:
#                                                           version
# -----------------------------------------------------------------------------------------------------------------------------
#  PostgreSQL 15.5 (Debian 15.5-1.pgdg120+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 12.2.0-14) 12.2.0, 64-bit
# (1 row)

# ✅ Shows PostgreSQL 15.x version
# ❌ If error "could not connect": Check container is running

echo ""
echo "Testing NATS connection..."
curl -s http://localhost:8222/varz | jq '.' | head -20

# Expected output (JSON with NATS stats):
# {
#   "server_id": "NDJZ...",
#   "server_name": "NDJZ...",
#   "version": "2.10.0",
#   "proto": 1,
#   "git_commit": "...",
#   "go": "go1.21.3",
#   "host": "0.0.0.0",
#   "port": 4222,
#   "max_connections": 65536,
#   "ping_interval": 120000000000,
#   "ping_max": 2,
#   "http_host": "0.0.0.0",
#   "http_port": 8222,
#   "https_port": 0,
#   "auth_timeout": 1,
#   "max_control_line": 4096,
#   ...
# }

# ✅ Shows JSON with server stats and version 2.10.x
# ❌ If "Connection refused": NATS container not running
# Note: Install jq if not available: sudo apt install -y jq

echo ""
echo "Testing Redis connection..."
sudo docker exec -it metabridge-redis redis-cli ping

# Expected output:
# PONG

# ✅ Responds with "PONG"
# ❌ If "Could not connect": Redis container not running

echo ""
echo "Testing Redis data operations..."
sudo docker exec -it metabridge-redis redis-cli SET test_key "test_value"
sudo docker exec -it metabridge-redis redis-cli GET test_key
sudo docker exec -it metabridge-redis redis-cli DEL test_key

# Expected output:
# OK
# "test_value"
# (integer) 1

# ✅ Redis can store and retrieve data
# ❌ If errors: Check Redis logs

echo ""
echo "✅ All infrastructure services are healthy!"
echo ""
```

### Test 2: API Health Checks

These tests verify that your API server is running and responding to requests correctly.

```bash
echo "========================================="
echo "Test 2: API Health Checks"
echo "========================================="
echo ""

# Test 1: Basic health endpoint
echo "Testing basic health endpoint..."
curl -s http://159.65.73.133:8080/health | jq '.'

# Expected output:
# {
#   "status": "healthy",
#   "timestamp": "2025-11-22T14:30:45Z",
#   "version": "1.0.0",
#   "uptime": 3600
# }

# ✅ Status is "healthy"
# ✅ Returns valid JSON
# ✅ Timestamp is current
# ❌ If "Connection refused": API server not running
# ❌ If HTML error page: Wrong port or nginx issue
# ❌ If timeout: Firewall blocking port 8080

echo ""
echo "Testing with verbose output for debugging..."
curl -v http://159.65.73.133:8080/health 2>&1 | grep -E '(HTTP|status)'

# Expected output:
# > GET /health HTTP/1.1
# < HTTP/1.1 200 OK
# < Content-Type: application/json
# {"status":"healthy",...}

# ✅ HTTP/1.1 200 OK response
# ❌ HTTP/1.1 404 Not Found: Route not configured
# ❌ HTTP/1.1 500 Internal Server Error: Server crash

echo ""
echo "Testing detailed API status..."
curl -s http://159.65.73.133:8080/v1/status | jq '.'

# Expected output:
# {
#   "api": {
#     "status": "healthy",
#     "version": "1.0.0",
#     "uptime_seconds": 3600
#   },
#   "database": {
#     "status": "connected",
#     "type": "postgresql",
#     "ping_ms": 2
#   },
#   "nats": {
#     "status": "connected",
#     "url": "nats://localhost:4222",
#     "servers": 1
#   },
#   "redis": {
#     "status": "connected",
#     "ping_ms": 1
#   }
# }

# ✅ All services show "connected" or "healthy"
# ❌ If any service shows "disconnected": Check that service
# ❌ If database ping_ms > 100: Database performance issue

echo ""
echo "Testing chain connectivity..."
curl -s http://159.65.73.133:8080/v1/chains/status | jq '.'

# Expected output:
# {
#   "ethereum": {
#     "chain_id": 11155111,
#     "name": "Ethereum Sepolia",
#     "type": "evm",
#     "healthy": true,
#     "rpc_url": "https://sepolia.infura.io/v3/...",
#     "block_number": 5234567,
#     "last_check": "2025-11-22T14:30:45Z",
#     "latency_ms": 245
#   },
#   "polygon": {
#     "chain_id": 80002,
#     "name": "Polygon Amoy",
#     "type": "evm",
#     "healthy": true,
#     "rpc_url": "https://rpc-amoy.polygon.technology/",
#     "block_number": 12345678,
#     "last_check": "2025-11-22T14:30:45Z",
#     "latency_ms": 189
#   },
#   "bnb": {
#     "chain_id": 97,
#     "name": "BNB Testnet",
#     "type": "evm",
#     "healthy": true,
#     "block_number": 34567890,
#     "latency_ms": 156
#   },
#   "avalanche": {
#     "chain_id": 43113,
#     "name": "Avalanche Fuji",
#     "type": "evm",
#     "healthy": true,
#     "block_number": 23456789,
#     "latency_ms": 203
#   },
#   "solana": {
#     "name": "Solana Devnet",
#     "type": "solana",
#     "healthy": true,
#     "rpc_url": "https://api.devnet.solana.com",
#     "slot": 287654321,
#     "latency_ms": 178
#   },
#   "near": {
#     "name": "NEAR Testnet",
#     "type": "near",
#     "healthy": true,
#     "rpc_url": "https://rpc.testnet.near.org",
#     "block_height": 123456789,
#     "latency_ms": 312
#   }
# }

# ✅ All chains show "healthy": true
# ✅ Block numbers/slots are recent
# ✅ Latency < 500ms (acceptable < 1000ms)
# ⚠️ If healthy: false - RPC endpoint down or API key issue
# ⚠️ If high latency (>1000ms) - Network congestion or slow RPC

echo ""
echo "Testing bridge statistics..."
curl -s http://159.65.73.133:8080/v1/stats | jq '.'

# Expected output (fresh deployment):
# {
#   "total_messages": 0,
#   "pending_messages": 0,
#   "processing_messages": 0,
#   "completed_messages": 0,
#   "failed_messages": 0,
#   "total_volume_usd": "0",
#   "total_fees_usd": "0",
#   "success_rate": 0,
#   "average_processing_time_seconds": 0,
#   "chains": {
#     "ethereum": {"sent": 0, "received": 0},
#     "polygon": {"sent": 0, "received": 0},
#     "bnb": {"sent": 0, "received": 0},
#     "avalanche": {"sent": 0, "received": 0},
#     "solana": {"sent": 0, "received": 0},
#     "near": {"sent": 0, "received": 0}
#   }
# }

# ✅ Returns valid stats structure
# ✅ All values are 0 for fresh deployment
# ❌ If error: Database connection issue

echo ""
echo "Testing API response times..."
time curl -s http://159.65.73.133:8080/health > /dev/null

# Expected output:
# real    0m0.052s
# user    0m0.012s
# sys     0m0.008s

# ✅ Response time < 100ms is excellent
# ✅ Response time < 500ms is acceptable
# ⚠️ Response time > 1s indicates performance issues

echo ""
echo "Testing CORS headers..."
curl -s -I http://159.65.73.133:8080/health | grep -i 'access-control'

# Expected output:
# Access-Control-Allow-Origin: *
# Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
# Access-Control-Allow-Headers: Content-Type, Authorization

# ✅ CORS headers present (needed for web frontends)
# ⚠️ If missing: Check .env.production CORS settings

echo ""
echo "✅ All API health checks passed!"
echo ""
```

### Test 3: Authentication Tests

```bash
# Login test (if auth enabled)
curl -X POST http://159.65.73.133:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@metabridge.local",
    "password": "admin123"
  }'

# Expected: JWT token in response
# {
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "user": {
#     "id": "admin-001",
#     "email": "admin@metabridge.local",
#     "role": "admin"
#   }
# }

# Save the token for later use
export JWT_TOKEN="<your_token_here>"

# Test authenticated endpoint
curl http://159.65.73.133:8080/v1/admin/users \
  -H "Authorization: Bearer $JWT_TOKEN"

# Expected: List of users
```

### Test 4: Database Tests

```bash
# Test 1: Check all tables exist
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_name;
EOF

# Expected tables:
# - messages
# - batches
# - batch_messages
# - routes
# - webhooks
# - users
# - api_keys
# - sessions

# Test 2: Verify admin user exists
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
SELECT id, email, role, active FROM users;
EOF

# Expected: admin-001 | admin@metabridge.local | admin | true

# Test 3: Check database size
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT pg_size_pretty(pg_database_size('metabridge_production')) as size;"

# Expected: Database size (should be ~50MB for fresh install)
```

### Test 5: Service Monitoring Tests

```bash
# Check systemd service status
sudo systemctl status metabridge-api --no-pager
sudo systemctl status metabridge-relayer --no-pager

# Both should show:
# Active: active (running)

# Check service logs for errors
sudo journalctl -u metabridge-api --since "5 minutes ago" --no-pager | grep -i error

# Expected: No critical errors (some warnings are normal)

# Check resource usage
ps aux | grep metabridge

# Expected: See api and relayer processes running

# Check memory usage
free -h

# Expected: At least 1GB free memory

# Check disk usage
df -h

# Expected: At least 10GB free disk space
```

### Test 6: Network Connectivity Tests

```bash
# Test 1: Check open ports
sudo netstat -tlnp | grep -E '(8080|5432|4222|6379)'

# Expected ports:
# - 8080  (API)
# - 5432  (PostgreSQL)
# - 4222  (NATS)
# - 6379  (Redis)

# Test 2: Test external API access
curl -I http://159.65.73.133:8080/health

# Expected:
# HTTP/1.1 200 OK
# Content-Type: application/json

# Test 3: Test firewall rules
sudo ufw status numbered

# Expected rules:
# [1] 22/tcp                     ALLOW IN    Anywhere
# [2] 80/tcp                     ALLOW IN    Anywhere
# [3] 443/tcp                    ALLOW IN    Anywhere
# [4] 8080/tcp                   ALLOW IN    Anywhere
```

### Test 7: End-to-End Bridge Flow Test (Optional)

```bash
# This test requires deployed smart contracts and test tokens

# Test 1: Create a bridge request
curl -X POST http://159.65.73.133:8080/v1/bridge/request \
  -H "Content-Type: application/json" \
  -d '{
    "source_chain": "polygon",
    "destination_chain": "bnb",
    "token_address": "0x...",
    "amount": "1000000000000000000",
    "recipient": "0x...",
    "sender": "0x..."
  }'

# Expected: Bridge request ID

# Test 2: Check request status
curl http://159.65.73.133:8080/v1/bridge/request/<request_id>

# Expected: Request details with status

# Test 3: List all bridge requests
curl http://159.65.73.133:8080/v1/messages?limit=10

# Expected: List of bridge messages
```

### Test 8: Performance Tests

```bash
# Test 1: API response time
time curl http://159.65.73.133:8080/health

# Expected: < 100ms

# Test 2: Database query performance
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
\timing on
SELECT COUNT(*) FROM messages;
EOF

# Expected: Query time < 10ms

# Test 3: Concurrent requests test (requires Apache Bench)
sudo apt install -y apache2-utils

ab -n 100 -c 10 http://159.65.73.133:8080/health

# Expected:
# - 100% successful requests
# - Average response time < 100ms
# - No failed requests
```

### Test 9: Log Analysis

```bash
# Check API logs for startup messages
sudo journalctl -u metabridge-api --since "10 minutes ago" --no-pager | head -50

# Expected to see:
# - "Starting Metabridge API..."
# - "Database connected"
# - "NATS connected"
# - "Redis connected"
# - "Server listening on :8080"

# Check for any error patterns
sudo journalctl -u metabridge-api --since "1 hour ago" --no-pager | grep -iE '(error|fatal|panic|failed)' | wc -l

# Expected: 0 or very low number

# Check relayer logs
sudo journalctl -u metabridge-relayer --since "10 minutes ago" --no-pager | head -50

# Expected to see:
# - "Relayer starting..."
# - "Connected to NATS"
# - "Listening for messages..."
```

### Test 10: Backup & Recovery Test

```bash
# Test 1: Create a manual backup
sudo docker exec metabridge-postgres pg_dump -U bridge_user metabridge_production > ~/test_backup_$(date +%Y%m%d).sql

# Verify backup file was created
ls -lh ~/test_backup_*.sql

# Expected: Backup file with size > 0 bytes

# Test 2: Insert test data
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
VALUES (
  'test-user-001',
  'test@metabridge.local',
  'Test User',
  '$2a$10$test',
  'user',
  true,
  NOW(),
  NOW()
);
EOF

# Verify insertion
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT COUNT(*) FROM users WHERE id='test-user-001';"

# Expected: 1

# Test 3: Test restore (to verify backup integrity)
# First, delete the test user
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "DELETE FROM users WHERE id='test-user-001';"

# Restore from backup
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < ~/test_backup_*.sql

# Verify restore
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT COUNT(*) FROM users;"

# Expected: Original count + test user

# Clean up test data
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "DELETE FROM users WHERE id='test-user-001';"
rm ~/test_backup_*.sql
```

## Step 17: View Logs

```bash
# View API logs (live)
sudo journalctl -u metabridge-api -f

# View relayer logs (live)
sudo journalctl -u metabridge-relayer -f

# View last 100 lines of API logs
sudo journalctl -u metabridge-api -n 100 --no-pager

# View logs with errors only
sudo journalctl -u metabridge-api --since "10 minutes ago" | grep -i error

# View Docker container logs
sudo docker compose -f docker-compose.production.yaml logs -f
```

## Step 18: Install Nginx (Optional - for production)

```bash
# Install Nginx
sudo apt install -y nginx

# Create Nginx configuration
sudo tee /etc/nginx/sites-available/metabridge > /dev/null << 'EOF'
server {
    listen 80;
    server_name 159.65.73.133;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/metabridge /etc/nginx/sites-enabled/

# Remove default site
sudo rm /etc/nginx/sites-enabled/default

# Test Nginx configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx

# Now you can access via: http://159.65.73.133
curl http://159.65.73.133/health
```

## Step 19: Set Up SSL (Optional - if you have a domain)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get SSL certificate (replace yourdomain.com with your actual domain)
sudo certbot --nginx -d api.yourdomain.com

# Auto-renewal is configured automatically
# Test auto-renewal
sudo certbot renew --dry-run
```

## Verification Checklist

Run these checks to ensure everything is working:

```bash
# ✅ Check Docker containers
sudo docker compose -f docker-compose.production.yaml ps
# All should be "Up"

# ✅ Check systemd services
sudo systemctl status metabridge-api
sudo systemctl status metabridge-relayer
# Both should be "active (running)"

# ✅ Check API health
curl http://159.65.73.133:8080/health
# Should return: {"status":"ok","version":"1.0.0"}

# ✅ Check database
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT COUNT(*) FROM users;"
# Should return: 1

# ✅ Check disk space
df -h
# Should have >10GB free

# ✅ Check memory
free -h
# Should have >1GB free

# ✅ Check firewall
sudo ufw status
# Should show ports 22, 80, 443, 8080 allowed
```

## Common Commands

```bash
# Restart API server
sudo systemctl restart metabridge-api

# Restart relayer
sudo systemctl restart metabridge-relayer

# Restart all Docker services
cd ~/projects/metabridge-engine-hub
sudo docker compose -f docker-compose.production.yaml restart

# View live logs
sudo journalctl -u metabridge-api -f

# Update code
cd ~/projects/metabridge-engine-hub
git pull origin main
go build -o bin/metabridge-api cmd/api/main.go
sudo systemctl restart metabridge-api

# Database backup
sudo docker exec metabridge-postgres pg_dump -U bridge_user metabridge_production > ~/backup_$(date +%Y%m%d).sql

# Restore database
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < ~/backup_20240101.sql
```

## Troubleshooting

### Service won't start

```bash
# Check logs
sudo journalctl -u metabridge-api -n 100 --no-pager

# Check if port is already in use
sudo lsof -i :8080

# Check environment file
cat ~/projects/metabridge-engine-hub/.env.production
```

### Database connection failed

```bash
# Check if PostgreSQL is running
sudo docker compose -f docker-compose.production.yaml ps postgres

# Test connection
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT 1;"

# Check password in .env.production matches the one used in Step 11
```

### Port already in use

```bash
# Find what's using port 8080
sudo lsof -i :8080

# Kill the process
sudo kill -9 <PID>

# Or change the port in .env.production
nano ~/projects/metabridge-engine-hub/.env.production
# Change SERVER_PORT=8080 to SERVER_PORT=8081
```

### Out of disk space

```bash
# Check disk usage
df -h

# Clean up Docker
sudo docker system prune -a --volumes

# Clean up old logs
sudo journalctl --vacuum-time=7d
```

## Next Steps

1. **Get API Keys** (if you haven't already):
   - Alchemy: https://www.alchemy.com/
   - Infura: https://infura.io/
   - Update `.env.production` with your keys

2. **Deploy Smart Contracts** (from your local machine or the droplet):
   - Follow the contract deployment guide in README.md
   - Update contract addresses in `.env.production`

3. **Enable Authentication** (optional):
   - Change `REQUIRE_AUTH=true` in `.env.production`
   - Restart API: `sudo systemctl restart metabridge-api`

4. **Set Up Domain** (optional):
   - Point your domain to `159.65.73.133`
   - Follow Step 19 to set up SSL

5. **Monitor Your Bridge**:
   - Access: http://159.65.73.133:8080/v1/stats
   - Check logs: `sudo journalctl -u metabridge-api -f`

## Your Deployment Summary

**Access Points**:
- API: `http://159.65.73.133:8080`
- Health: `http://159.65.73.133:8080/health`
- Chain Status: `http://159.65.73.133:8080/v1/chains/status`
- Stats: `http://159.65.73.133:8080/v1/stats`

**Admin Credentials**:
- Email: `admin@metabridge.local`
- Password: `admin123` (or whatever you set in Step 12)

**Services**:
- API Server: `sudo systemctl status metabridge-api`
- Relayer: `sudo systemctl status metabridge-relayer`
- PostgreSQL: `sudo docker ps | grep postgres`
- NATS: `sudo docker ps | grep nats`
- Redis: `sudo docker ps | grep redis`

**Important Files**:
- Environment: `~/projects/metabridge-engine-hub/.env.production`
- Binaries: `~/projects/metabridge-engine-hub/bin/`
- Logs: `sudo journalctl -u metabridge-api`

🎉 **Congratulations! Your Metabridge is now running on DigitalOcean!**
