package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
)

// VerifyECDSASignature verifies an ECDSA signature for EVM chains
func VerifyECDSASignature(messageHash []byte, signature string, address string) error {
	// Decode signature from hex
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Signature should be 65 bytes (R + S + V)
	if len(sigBytes) != 65 {
		return fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Recover public key from signature
	// Note: Ethereum signatures have V as the last byte
	// We need to adjust V for ecrecover
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(messageHash, sigBytes)
	if err != nil {
		return fmt.Errorf("failed to recover public key: %w", err)
	}

	// Get address from public key
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// Compare with expected address
	expectedAddr := common.HexToAddress(address)
	if recoveredAddr != expectedAddr {
		return fmt.Errorf("signature verification failed: expected %s, got %s",
			expectedAddr.Hex(), recoveredAddr.Hex())
	}

	return nil
}

// VerifyEd25519Signature verifies an Ed25519 signature for Solana/NEAR chains
func VerifyEd25519Signature(messageHash []byte, signature string, publicKeyStr string) error {
	// Decode signature
	var sigBytes []byte
	var err error

	// Try base58 first (Solana)
	sigBytes, err = base58.Decode(signature)
	if err != nil {
		// Try hex (NEAR)
		sigBytes, err = hex.DecodeString(signature)
		if err != nil {
			return fmt.Errorf("failed to decode signature: %w", err)
		}
	}

	// Signature should be 64 bytes for Ed25519
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Decode public key
	var pubKeyBytes []byte

	// Try base58 first (Solana)
	pubKeyBytes, err = base58.Decode(publicKeyStr)
	if err != nil {
		// Try hex (NEAR)
		pubKeyBytes, err = hex.DecodeString(publicKeyStr)
		if err != nil {
			return fmt.Errorf("failed to decode public key: %w", err)
		}
	}

	// Public key should be 32 bytes
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length: %d", len(pubKeyBytes))
	}

	// Verify signature
	pubKey := ed25519.PublicKey(pubKeyBytes)
	if !ed25519.Verify(pubKey, messageHash, sigBytes) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// Keccak256 computes the Keccak256 hash of the input data
func Keccak256(data []byte) []byte {
	return crypto.Keccak256(data)
}

// Keccak256Hash computes the Keccak256 hash and returns it as a common.Hash
func Keccak256Hash(data []byte) common.Hash {
	return crypto.Keccak256Hash(data)
}
