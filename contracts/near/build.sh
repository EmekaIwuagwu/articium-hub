#!/bin/bash
set -e

# Build script for NEAR bridge contract

echo "Building NEAR bridge contract..."

# Set target directory
TARGET_DIR="../../target"
WASM_DIR="$TARGET_DIR/wasm32-unknown-unknown/release"

# Clean previous builds
cargo clean

# Build the contract
RUSTFLAGS='-C link-arg=-s' cargo build --target wasm32-unknown-unknown --release

# Create output directory
mkdir -p ./res

# Copy wasm file
cp $WASM_DIR/near_bridge.wasm ./res/

# Get file size
wasm_size=$(stat -f%z "./res/near_bridge.wasm" 2>/dev/null || stat -c%s "./res/near_bridge.wasm")

echo "✓ Contract built successfully!"
echo "  WASM size: $wasm_size bytes"
echo "  Output: ./res/near_bridge.wasm"

# Optional: Strip and optimize if wasm-opt is available
if command -v wasm-opt &> /dev/null; then
    echo "Optimizing WASM with wasm-opt..."
    wasm-opt -Oz ./res/near_bridge.wasm -o ./res/near_bridge.wasm
    optimized_size=$(stat -f%z "./res/near_bridge.wasm" 2>/dev/null || stat -c%s "./res/near_bridge.wasm")
    echo "✓ Optimized size: $optimized_size bytes"
fi

echo "✓ Build complete!"
