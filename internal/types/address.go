package types

import (
	"fmt"
	"strings"
)

// AddressFormat represents the format of a blockchain address
type AddressFormat string

const (
	AddressFormatEVM    AddressFormat = "EVM"    // 0x... (40 hex chars)
	AddressFormatBase58 AddressFormat = "BASE58" // Solana format
	AddressFormatNamed  AddressFormat = "NAMED"  // NEAR account.testnet
)

// Address represents a cross-chain address
type Address struct {
	Raw       string        `json:"raw" db:"raw"`
	ChainType ChainType     `json:"chain_type" db:"chain_type"`
	Format    AddressFormat `json:"format" db:"format"`
}

// NewAddress creates a new Address with validation
func NewAddress(raw string, chainType ChainType) (Address, error) {
	addr := Address{
		Raw:       raw,
		ChainType: chainType,
	}

	// Determine format and validate
	switch chainType {
	case ChainTypeEVM:
		if err := validateEVMAddress(raw); err != nil {
			return Address{}, err
		}
		addr.Format = AddressFormatEVM

	case ChainTypeSolana:
		if err := validateSolanaAddress(raw); err != nil {
			return Address{}, err
		}
		addr.Format = AddressFormatBase58

	case ChainTypeNEAR:
		if err := validateNEARAddress(raw); err != nil {
			return Address{}, err
		}
		addr.Format = AddressFormatNamed

	default:
		return Address{}, fmt.Errorf("unsupported chain type: %s", chainType)
	}

	return addr, nil
}

// String returns the string representation of the address
func (a Address) String() string {
	return a.Raw
}

// Equals checks if two addresses are equal
func (a Address) Equals(other Address) bool {
	return strings.EqualFold(a.Raw, other.Raw) && a.ChainType == other.ChainType
}

// validateEVMAddress validates an EVM address format
func validateEVMAddress(addr string) error {
	if !strings.HasPrefix(addr, "0x") {
		return fmt.Errorf("EVM address must start with 0x")
	}
	if len(addr) != 42 {
		return fmt.Errorf("EVM address must be 42 characters (0x + 40 hex)")
	}
	// Check if all characters after 0x are valid hex
	for _, c := range addr[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("EVM address contains invalid hex character: %c", c)
		}
	}
	return nil
}

// validateSolanaAddress validates a Solana address (base58 encoded public key)
func validateSolanaAddress(addr string) error {
	if len(addr) < 32 || len(addr) > 44 {
		return fmt.Errorf("Solana address must be 32-44 characters in base58")
	}
	// Check for valid base58 characters
	for _, c := range addr {
		if !isBase58Char(c) {
			return fmt.Errorf("Solana address contains invalid base58 character: %c", c)
		}
	}
	return nil
}

// validateNEARAddress validates a NEAR address format
func validateNEARAddress(addr string) error {
	if len(addr) < 2 || len(addr) > 64 {
		return fmt.Errorf("NEAR address must be 2-64 characters")
	}

	// NEAR addresses can be:
	// 1. Named accounts: alice.near, alice.testnet
	// 2. Implicit accounts: 64 hex characters

	// Check if it's an implicit account (64 hex chars)
	if len(addr) == 64 {
		for _, c := range addr {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("NEAR implicit account must be 64 lowercase hex characters")
			}
		}
		return nil
	}

	// Check named account format
	// Can contain: lowercase letters, digits, _ and -
	// Must end with .near, .testnet, or another valid TLD
	for i, c := range addr {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
			return fmt.Errorf("NEAR named account contains invalid character at position %d: %c", i, c)
		}
	}

	// Should contain at least one dot for named accounts (except implicit)
	if !strings.Contains(addr, ".") && len(addr) != 64 {
		return fmt.Errorf("NEAR named account should contain a dot (e.g., account.near)")
	}

	return nil
}

// isBase58Char checks if a character is valid in base58 encoding
func isBase58Char(c rune) bool {
	// Base58 alphabet: 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
	// (excludes 0, O, I, l to avoid confusion)
	return (c >= '1' && c <= '9') ||
		(c >= 'A' && c <= 'H') ||
		(c >= 'J' && c <= 'N') ||
		(c >= 'P' && c <= 'Z') ||
		(c >= 'a' && c <= 'k') ||
		(c >= 'm' && c <= 'z')
}
