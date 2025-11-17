package types

import (
	"fmt"
)

// SignatureScheme represents the cryptographic signature scheme
type SignatureScheme string

const (
	SignatureSchemeECDSA   SignatureScheme = "ECDSA"   // secp256k1 for EVM
	SignatureSchemeEd25519 SignatureScheme = "Ed25519" // for Solana and NEAR
)

// Signature represents a cryptographic signature
type Signature struct {
	Scheme    SignatureScheme `json:"scheme"`
	Data      []byte          `json:"data"`
	PublicKey []byte          `json:"public_key"`
	Signer    string          `json:"signer"` // Address of signer
}

// NewSignature creates a new signature
func NewSignature(scheme SignatureScheme, data []byte, publicKey []byte, signer string) *Signature {
	return &Signature{
		Scheme:    scheme,
		Data:      data,
		PublicKey: publicKey,
		Signer:    signer,
	}
}

// Validate performs basic validation on the signature
func (s *Signature) Validate() error {
	if s.Scheme == "" {
		return fmt.Errorf("signature scheme cannot be empty")
	}

	if len(s.Data) == 0 {
		return fmt.Errorf("signature data cannot be empty")
	}

	switch s.Scheme {
	case SignatureSchemeECDSA:
		// ECDSA signatures are typically 65 bytes (r + s + v)
		if len(s.Data) != 65 {
			return fmt.Errorf("ECDSA signature must be 65 bytes, got %d", len(s.Data))
		}

	case SignatureSchemeEd25519:
		// Ed25519 signatures are 64 bytes
		if len(s.Data) != 64 {
			return fmt.Errorf("Ed25519 signature must be 64 bytes, got %d", len(s.Data))
		}

	default:
		return fmt.Errorf("unknown signature scheme: %s", s.Scheme)
	}

	return nil
}

// GetSchemeForChain returns the appropriate signature scheme for a chain type
func GetSchemeForChain(chainType ChainType) (SignatureScheme, error) {
	switch chainType {
	case ChainTypeEVM:
		return SignatureSchemeECDSA, nil
	case ChainTypeSolana, ChainTypeNEAR:
		return SignatureSchemeEd25519, nil
	default:
		return "", fmt.Errorf("unknown chain type: %s", chainType)
	}
}
