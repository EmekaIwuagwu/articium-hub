package relayer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/EmekaIwuagwu/metabridge-hub/internal/blockchain"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/config"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/crypto"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/database"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/monitoring"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/security"
	"github.com/EmekaIwuagwu/metabridge-hub/internal/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog"
)

// Processor processes cross-chain messages and broadcasts them to destination chains
type Processor struct {
	clients   map[string]types.UniversalClient
	signers   map[string]crypto.UniversalSigner
	db        *database.DB
	config    *config.Config
	validator *security.Validator
	logger    zerolog.Logger
	chainCfg  map[string]*types.ChainConfig
}

// NewProcessor creates a new message processor
func NewProcessor(
	clients map[string]types.UniversalClient,
	signers map[string]crypto.UniversalSigner,
	db *database.DB,
	cfg *config.Config,
	validator *security.Validator,
	logger zerolog.Logger,
) *Processor {
	chainCfg := make(map[string]*types.ChainConfig)
	for _, chain := range cfg.Chains {
		chainCfg[chain.Name] = &chain
	}

	return &Processor{
		clients:   clients,
		signers:   signers,
		db:        db,
		config:    cfg,
		validator: validator,
		logger:    logger.With().Str("component", "processor").Logger(),
		chainCfg:  chainCfg,
	}
}

// ProcessMessage processes a cross-chain message
func (p *Processor) ProcessMessage(ctx context.Context, msg *types.CrossChainMessage) error {
	startTime := time.Now()
	defer func() {
		monitoring.RelayerMessageProcessingDuration.WithLabelValues(
			msg.SourceChain.Name,
			msg.DestinationChain.Name,
		).Observe(time.Since(startTime).Seconds())
	}()

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("source", msg.SourceChain.Name).
		Str("destination", msg.DestinationChain.Name).
		Str("type", string(msg.Type)).
		Msg("Processing message")

	// Validate message security
	if err := p.validator.ValidateMessage(ctx, msg); err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Message failed security validation")
		monitoring.RecordMessageProcessingError("security_validation", msg.SourceChain.Name)
		return fmt.Errorf("security validation failed: %w", err)
	}

	// Check if message already processed
	status, err := p.db.GetMessageStatus(ctx, msg.ID)
	if err == nil && status == types.MessageStatusCompleted {
		p.logger.Warn().
			Str("message_id", msg.ID).
			Msg("Message already processed, skipping")
		return nil
	}

	// Verify validator signatures
	if err := p.verifySignatures(ctx, msg); err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Signature verification failed")
		monitoring.RecordMessageProcessingError("signature_verification", msg.SourceChain.Name)
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Process based on destination chain type
	destClient, ok := p.clients[msg.DestinationChain.Name]
	if !ok {
		return fmt.Errorf("client not found for chain: %s", msg.DestinationChain.Name)
	}

	var txHash string
	switch destClient.GetChainType() {
	case types.ChainTypeEVM:
		txHash, err = p.processEVMMessage(ctx, msg, destClient)
	case types.ChainTypeSolana:
		txHash, err = p.processSolanaMessage(ctx, msg, destClient)
	case types.ChainTypeNEAR:
		txHash, err = p.processNEARMessage(ctx, msg, destClient)
	default:
		return fmt.Errorf("unsupported chain type: %s", destClient.GetChainType())
	}

	if err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Failed to broadcast transaction")
		monitoring.RecordMessageProcessingError("transaction_broadcast", msg.DestinationChain.Name)
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Update message status
	if err := p.db.UpdateMessageStatus(ctx, msg.ID, types.MessageStatusCompleted, txHash); err != nil {
		p.logger.Error().
			Err(err).
			Str("message_id", msg.ID).
			Msg("Failed to update message status")
		// Don't return error - transaction was broadcast successfully
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Str("tx_hash", txHash).
		Str("destination", msg.DestinationChain.Name).
		Msg("Message processed successfully")

	monitoring.RecordMessageProcessed(msg.SourceChain.Name, msg.DestinationChain.Name, string(msg.Type))
	return nil
}

// verifySignatures verifies validator signatures on the message
func (p *Processor) verifySignatures(ctx context.Context, msg *types.CrossChainMessage) error {
	// Get required signature threshold based on environment
	requiredSigs := p.config.Security.RequiredSignatures

	if len(msg.ValidatorSignatures) < requiredSigs {
		return fmt.Errorf("insufficient signatures: got %d, need %d",
			len(msg.ValidatorSignatures), requiredSigs)
	}

	// Verify each signature
	validSigs := 0
	seenValidators := make(map[string]bool)

	for _, sig := range msg.ValidatorSignatures {
		// Check for duplicate validators
		if seenValidators[sig.ValidatorAddress] {
			p.logger.Warn().
				Str("validator", sig.ValidatorAddress).
				Msg("Duplicate signature from validator")
			continue
		}
		seenValidators[sig.ValidatorAddress] = true

		// Verify signature
		if err := p.verifyValidatorSignature(ctx, msg, &sig); err != nil {
			p.logger.Warn().
				Err(err).
				Str("validator", sig.ValidatorAddress).
				Msg("Invalid signature from validator")
			continue
		}

		validSigs++
	}

	if validSigs < requiredSigs {
		return fmt.Errorf("insufficient valid signatures: got %d, need %d",
			validSigs, requiredSigs)
	}

	p.logger.Info().
		Str("message_id", msg.ID).
		Int("valid_signatures", validSigs).
		Int("required", requiredSigs).
		Msg("Signature verification passed")

	return nil
}

// verifyValidatorSignature verifies a single validator signature
func (p *Processor) verifyValidatorSignature(ctx context.Context, msg *types.CrossChainMessage, sig *types.ValidatorSignature) error {
	// Create message hash for verification
	msgHash, err := p.createMessageHash(msg)
	if err != nil {
		return fmt.Errorf("failed to create message hash: %w", err)
	}

	// Get validator's chain type to determine signature scheme
	// For simplicity, we'll use the source chain's type
	// In production, you might want to use a specific validator registry
	sourceClient := p.clients[msg.SourceChain.Name]
	chainType := sourceClient.GetChainType()

	// Verify signature based on chain type
	switch chainType {
	case types.ChainTypeEVM:
		return crypto.VerifyECDSASignature(msgHash, sig.Signature, sig.ValidatorAddress)
	case types.ChainTypeSolana, types.ChainTypeNEAR:
		return crypto.VerifyEd25519Signature(msgHash, sig.Signature, sig.ValidatorAddress)
	default:
		return fmt.Errorf("unsupported chain type for signature verification")
	}
}

// createMessageHash creates a deterministic hash of the message for signing
func (p *Processor) createMessageHash(msg *types.CrossChainMessage) ([]byte, error) {
	// Create a canonical representation of the message
	data := struct {
		ID               string
		Type             types.MessageType
		SourceChainID    string
		DestChainID      string
		Sender           string
		Recipient        string
		Payload          json.RawMessage
		Nonce            uint64
		Timestamp        int64
	}{
		ID:               msg.ID,
		Type:             msg.Type,
		SourceChainID:    msg.SourceChain.ChainID,
		DestChainID:      msg.DestinationChain.ChainID,
		Sender:           msg.Sender.Raw,
		Recipient:        msg.Recipient.Raw,
		Payload:          msg.Payload,
		Nonce:            msg.Nonce,
		Timestamp:        msg.Timestamp,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return crypto.Keccak256(jsonData), nil
}

// processEVMMessage processes a message for EVM chains
func (p *Processor) processEVMMessage(ctx context.Context, msg *types.CrossChainMessage, client types.UniversalClient) (string, error) {
	p.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Processing EVM message")

	// Get chain configuration
	chainCfg, ok := p.chainCfg[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("chain config not found: %s", msg.DestinationChain.Name)
	}

	// Get signer for this chain
	signer, ok := p.signers[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("signer not found for chain: %s", msg.DestinationChain.Name)
	}

	// Build transaction based on message type
	var tx *ethTypes.Transaction
	var err error

	switch msg.Type {
	case types.MessageTypeTokenTransfer:
		tx, err = p.buildEVMTokenUnlockTx(msg, chainCfg)
	case types.MessageTypeNFTTransfer:
		tx, err = p.buildEVMNFTUnlockTx(msg, chainCfg)
	default:
		return "", fmt.Errorf("unsupported message type: %s", msg.Type)
	}

	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	// Sign transaction
	signedTx, err := signer.SignTransaction(ctx, tx, msg.DestinationChain.ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Broadcast transaction
	txHash, err := client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for confirmation if needed
	if chainCfg.ConfirmationBlocks > 0 {
		p.logger.Debug().
			Str("tx_hash", txHash).
			Uint64("confirmations", chainCfg.ConfirmationBlocks).
			Msg("Waiting for transaction confirmation")

		// In production, you would implement proper confirmation waiting
		// For now, we'll just return the tx hash
	}

	return txHash, nil
}

// buildEVMTokenUnlockTx builds a token unlock transaction for EVM chains
func (p *Processor) buildEVMTokenUnlockTx(msg *types.CrossChainMessage, chainCfg *types.ChainConfig) (*ethTypes.Transaction, error) {
	// Parse payload
	var payload types.TokenTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Build contract call data
	// unlockToken(bytes32 messageId, address recipient, address token, uint256 amount, bytes[] signatures)

	contractABI, err := abi.JSON(nil) // In production, load actual bridge ABI
	if err != nil {
		return nil, err
	}

	// Prepare signature array
	signatures := make([][]byte, len(msg.ValidatorSignatures))
	for i, sig := range msg.ValidatorSignatures {
		signatures[i] = []byte(sig.Signature)
	}

	// Pack function call
	data, err := contractABI.Pack(
		"unlockToken",
		[32]byte{}, // messageId (convert msg.ID to bytes32)
		common.HexToAddress(msg.Recipient.Raw),
		common.HexToAddress(payload.TokenAddress),
		new(big.Int).SetUint64(payload.Amount),
		signatures,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create transaction
	// In production, you would:
	// 1. Get nonce from account
	// 2. Estimate gas
	// 3. Get current gas price
	// 4. Build proper transaction with all parameters

	tx := ethTypes.NewTransaction(
		0, // nonce - should be fetched
		common.HexToAddress(chainCfg.BridgeContract),
		big.NewInt(0), // value
		300000, // gas limit - should be estimated
		big.NewInt(20000000000), // gas price - should be fetched
		data,
	)

	return tx, nil
}

// buildEVMNFTUnlockTx builds an NFT unlock transaction for EVM chains
func (p *Processor) buildEVMNFTUnlockTx(msg *types.CrossChainMessage, chainCfg *types.ChainConfig) (*ethTypes.Transaction, error) {
	// Parse payload
	var payload types.NFTTransferPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Similar to token unlock but for NFTs
	// unlockNFT(bytes32 messageId, address recipient, address nftContract, uint256 tokenId, bytes[] signatures)

	// Implementation similar to buildEVMTokenUnlockTx
	// Returning placeholder for now
	return nil, fmt.Errorf("NFT unlock not fully implemented")
}

// processSolanaMessage processes a message for Solana
func (p *Processor) processSolanaMessage(ctx context.Context, msg *types.CrossChainMessage, client types.UniversalClient) (string, error) {
	p.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Processing Solana message")

	// Get signer for Solana
	signer, ok := p.signers[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("signer not found for Solana")
	}

	// Build Solana transaction
	// This would involve:
	// 1. Creating instruction to call unlock function on Solana bridge program
	// 2. Building transaction with proper accounts
	// 3. Signing and sending

	// Placeholder implementation
	_ = signer
	return "", fmt.Errorf("Solana message processing not fully implemented")
}

// processNEARMessage processes a message for NEAR
func (p *Processor) processNEARMessage(ctx context.Context, msg *types.CrossChainMessage, client types.UniversalClient) (string, error) {
	p.logger.Debug().
		Str("message_id", msg.ID).
		Msg("Processing NEAR message")

	// Get signer for NEAR
	signer, ok := p.signers[msg.DestinationChain.Name]
	if !ok {
		return "", fmt.Errorf("signer not found for NEAR")
	}

	// Build NEAR transaction
	// This would involve:
	// 1. Creating function call action to bridge contract
	// 2. Building transaction with proper parameters
	// 3. Signing and sending

	// Placeholder implementation
	_ = signer
	return "", fmt.Errorf("NEAR message processing not fully implemented")
}
