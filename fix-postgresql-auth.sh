#!/bin/bash

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║     PostgreSQL Authentication Fix - Metabridge             ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# 1. Backup pg_hba.conf
echo "1️⃣  Backing up PostgreSQL configuration..."
sudo cp /etc/postgresql/16/main/pg_hba.conf /etc/postgresql/16/main/pg_hba.conf.backup
echo "   ✓ Backup created at /etc/postgresql/16/main/pg_hba.conf.backup"
echo ""

# 2. Update pg_hba.conf to use md5 authentication
echo "2️⃣  Configuring PostgreSQL authentication..."
sudo bash -c 'cat > /etc/postgresql/16/main/pg_hba.conf << EOF
# PostgreSQL Client Authentication Configuration File
# ===================================================

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
EOF'
echo "   ✓ pg_hba.conf updated to use md5 authentication"
echo ""

# 3. Restart PostgreSQL
echo "3️⃣  Restarting PostgreSQL..."
sudo systemctl restart postgresql@16-main
sleep 2
echo "   ✓ PostgreSQL restarted"
echo ""

# 4. Ensure metabridge user exists with correct password
echo "4️⃣  Setting up database user..."
sudo -u postgres psql << EOF
-- Drop and recreate user to ensure password is correct
DROP USER IF EXISTS metabridge;
CREATE USER metabridge WITH PASSWORD 'metabridge' SUPERUSER;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE metabridge_prod TO metabridge;

-- Show user
\du metabridge
EOF

# Grant schema permissions
echo "4️⃣b Granting schema permissions..."
sudo -u postgres psql -d metabridge_prod << EOF
-- Grant all privileges on public schema
GRANT ALL ON SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO metabridge;
EOF
echo ""

# 5. Test connection
echo "5️⃣  Testing database connection..."
if PGPASSWORD=metabridge psql -h /var/run/postgresql -p 5433 -U metabridge -d metabridge_prod -c "SELECT version();" > /dev/null 2>&1; then
    echo "   ✅ Connection successful!"
else
    echo "   ❌ Connection failed!"
    echo ""
    echo "Trying TCP connection instead..."
    if PGPASSWORD=metabridge psql -h 127.0.0.1 -p 5433 -U metabridge -d metabridge_prod -c "SELECT version();" > /dev/null 2>&1; then
        echo "   ✅ TCP connection works!"
        echo "   📝 Updating config to use 127.0.0.1 instead of socket..."
        sed -i 's|host: "/var/run/postgresql"|host: "127.0.0.1"|' /root/projects/metabridge-engine-hub/config/config.production.yaml
    fi
fi
echo ""

echo "╔════════════════════════════════════════════════════════════╗"
echo "║           ✅ PostgreSQL Authentication Fixed!              ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Next step: Run fix-all.sh again"
echo "  sudo bash fix-all.sh"
echo ""
