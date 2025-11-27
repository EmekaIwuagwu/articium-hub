package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/gorilla/mux"
)

// Chain handlers

func (s *Server) handleListChains(w http.ResponseWriter, r *http.Request) {
	chains := make([]map[string]interface{}, 0)

	for _, client := range s.clients {
		info := client.GetChainInfo()
		chains = append(chains, map[string]interface{}{
			"name":        info.Name,
			"type":        info.Type,
			"chain_id":    info.ChainID,
			"network_id":  info.NetworkID,
			"environment": info.Environment,
			"healthy":     client.IsHealthy(r.Context()),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chains": chains,
		"total":  len(chains),
	})
}

func (s *Server) handleChainStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainName := vars["chain"]

	client, exists := s.clients[chainName]
	if !exists {
		respondError(w, http.StatusNotFound, "chain not found", nil)
		return
	}

	info := client.GetChainInfo()
	blockNumber, err := client.GetLatestBlockNumber(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get block number", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"name":          info.Name,
		"type":          info.Type,
		"chain_id":      info.ChainID,
		"network_id":    info.NetworkID,
		"environment":   info.Environment,
		"healthy":       client.IsHealthy(r.Context()),
		"block_number":  blockNumber,
		"block_time":    client.GetBlockTime().String(),
		"confirmations": client.GetConfirmationBlocks(),
	})
}

func (s *Server) handleAllChainsStatus(w http.ResponseWriter, r *http.Request) {
	status := make(map[string]interface{})

	for name, client := range s.clients {
		info := client.GetChainInfo()
		blockNumber, _ := client.GetLatestBlockNumber(r.Context())

		status[name] = map[string]interface{}{
			"healthy":      client.IsHealthy(r.Context()),
			"block_number": blockNumber,
			"chain_type":   info.Type,
		}
	}

	respondJSON(w, http.StatusOK, status)
}

// Bridge handlers

type BridgeTokenRequest struct {
	SourceChain      string `json:"source_chain"`
	DestinationChain string `json:"dest_chain"`
	TokenAddress     string `json:"token_address"`
	Amount           string `json:"amount"`
	Recipient        string `json:"recipient"`
	Sender           string `json:"sender,omitempty"`
}

func (s *Server) handleBridgeToken(w http.ResponseWriter, r *http.Request) {
	var req BridgeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.SourceChain == "" || req.DestinationChain == "" {
		respondError(w, http.StatusBadRequest, "source_chain and dest_chain are required", nil)
		return
	}

	if req.TokenAddress == "" || req.Amount == "" || req.Recipient == "" {
		respondError(w, http.StatusBadRequest, "token_address, amount, and recipient are required", nil)
		return
	}

	// Check if chains exist
	sourceClient, sourceExists := s.clients[req.SourceChain]
	if !sourceExists {
		respondError(w, http.StatusBadRequest, "invalid source chain", nil)
		return
	}
	destClient, destExists := s.clients[req.DestinationChain]
	if !destExists {
		respondError(w, http.StatusBadRequest, "invalid destination chain", nil)
		return
	}

	// Get chain info
	sourceChainInfo := sourceClient.GetChainInfo()
	destChainInfo := destClient.GetChainInfo()

	// Determine token standard based on source chain type
	var tokenStandard string
	switch sourceChainInfo.Type {
	case types.ChainTypeEVM:
		tokenStandard = "ERC20"
	case types.ChainTypeSolana:
		tokenStandard = "SPL"
	case types.ChainTypeNEAR:
		tokenStandard = "NEP141"
	default:
		tokenStandard = "UNKNOWN"
	}

	// Create payload
	payload := types.TokenTransferPayload{
		TokenAddress: types.Address{
			Raw:      req.TokenAddress,
			Type:     string(sourceChainInfo.Type),
			Standard: tokenStandard,
		},
		Amount:        req.Amount,
		TokenStandard: tokenStandard,
		Decimals:      18, // Default, should be fetched from token contract
	}

	// Create sender address (use provided or default)
	senderAddress := req.Sender
	if senderAddress == "" {
		senderAddress = "0x0000000000000000000000000000000000000000" // Placeholder
	}

	// Create cross-chain message
	msg, err := types.NewCrossChainMessage(
		types.MessageTypeTokenTransfer,
		sourceChainInfo,
		destChainInfo,
		types.Address{
			Raw:      senderAddress,
			Type:     string(sourceChainInfo.Type),
			Standard: tokenStandard,
		},
		types.Address{
			Raw:      req.Recipient,
			Type:     string(destChainInfo.Type),
			Standard: tokenStandard,
		},
		payload,
	)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create cross-chain message")
		respondError(w, http.StatusInternalServerError, "failed to create message", err)
		return
	}

	// Set required signatures based on config
	msg.RequiredSignatures = s.config.Security.RequiredSignatures

	// Save message to database
	if err := s.db.SaveMessage(r.Context(), msg); err != nil {
		s.logger.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to save message to database")
		respondError(w, http.StatusInternalServerError, "failed to save message", err)
		return
	}

	s.logger.Info().
		Str("message_id", msg.ID).
		Str("source", req.SourceChain).
		Str("destination", req.DestinationChain).
		Msg("Message saved to database")

	// Publish message to queue for processing
	if s.queue != nil {
		if err := s.queue.Publish(r.Context(), msg); err != nil {
			s.logger.Error().Err(err).Str("message_id", msg.ID).Msg("Failed to publish message to queue")
			// Don't return error - message is saved in DB and can be processed later
		} else {
			s.logger.Info().
				Str("message_id", msg.ID).
				Msg("Message published to queue")
		}
	} else {
		s.logger.Warn().Msg("Queue not available, message saved but not queued")
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":     "pending",
		"message":    "Bridge request received and queued for processing",
		"message_id": msg.ID,
		"request":    req,
	})
}

type BridgeNFTRequest struct {
	SourceChain      string `json:"source_chain"`
	DestinationChain string `json:"dest_chain"`
	NFTContract      string `json:"nft_contract"`
	TokenID          string `json:"token_id"`
	Recipient        string `json:"recipient"`
	Sender           string `json:"sender,omitempty"`
}

func (s *Server) handleBridgeNFT(w http.ResponseWriter, r *http.Request) {
	var req BridgeNFTRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate request
	if req.SourceChain == "" || req.DestinationChain == "" {
		respondError(w, http.StatusBadRequest, "source_chain and dest_chain are required", nil)
		return
	}

	if req.NFTContract == "" || req.TokenID == "" || req.Recipient == "" {
		respondError(w, http.StatusBadRequest, "nft_contract, token_id, and recipient are required", nil)
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "pending",
		"message": "NFT bridge request received and will be processed",
		"request": req,
	})
}

// Message handlers

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 50
	offset := 0
	status := r.URL.Query().Get("status")

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Query messages from database
	var messages []types.CrossChainMessage
	var err error

	if status != "" {
		messages, err = s.db.GetMessagesByStatus(r.Context(), types.MessageStatus(status), limit, offset)
	} else {
		// Get all recent messages (using completed status with high limit as fallback)
		// TODO: Add GetAllMessages method to database package for better performance
		messages, err = s.db.GetMessagesByStatus(r.Context(), types.MessageStatusCompleted, limit, offset)
	}

	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to query messages from database")
		respondError(w, http.StatusInternalServerError, "failed to retrieve messages", err)
		return
	}

	// Get total count
	totalPending, _ := s.db.GetPendingMessagesCount(r.Context())
	totalCompleted, _ := s.db.GetProcessedMessagesCount(r.Context())
	totalFailed, _ := s.db.GetFailedMessagesCount(r.Context())
	total := totalPending + totalCompleted + totalFailed

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	// Query message from database
	message, err := s.db.GetMessage(r.Context(), messageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "message not found", err)
		} else {
			s.logger.Error().Err(err).Str("message_id", messageID).Msg("Failed to get message")
			respondError(w, http.StatusInternalServerError, "failed to retrieve message", err)
		}
		return
	}

	// Get validator signatures
	signatures, err := s.db.GetValidatorSignatures(r.Context(), messageID)
	if err != nil {
		s.logger.Warn().Err(err).Str("message_id", messageID).Msg("Failed to get validator signatures")
		// Continue without signatures
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    message,
		"signatures": signatures,
	})
}

func (s *Server) handleMessageStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]

	// Query message status from database
	status, err := s.db.GetMessageStatus(r.Context(), messageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "message not found", err)
		} else {
			s.logger.Error().Err(err).Str("message_id", messageID).Msg("Failed to get message status")
			respondError(w, http.StatusInternalServerError, "failed to retrieve message status", err)
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message_id": messageID,
		"status":     status,
	})
}

// Statistics handlers

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// Get bridge statistics from database
	pendingCount, err := s.db.GetPendingMessagesCount(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get pending messages count")
		pendingCount = 0
	}

	completedCount, err := s.db.GetProcessedMessagesCount(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get completed messages count")
		completedCount = 0
	}

	failedCount, err := s.db.GetFailedMessagesCount(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get failed messages count")
		failedCount = 0
	}

	totalMessages := pendingCount + completedCount + failedCount

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total_messages":     totalMessages,
		"pending_messages":   pendingCount,
		"completed_messages": completedCount,
		"failed_messages":    failedCount,
		"total_volume_usd":   "0", // TODO: Implement volume tracking
		"supported_chains":   len(s.clients),
	})
}

func (s *Server) handleChainStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainName := vars["chain"]

	if _, exists := s.clients[chainName]; !exists {
		respondError(w, http.StatusNotFound, "chain not found", nil)
		return
	}

	// Get chain-specific statistics
	// Query messages where this chain is either source or destination
	limit := 1000 // High limit to get accurate count
	messagesFrom, err := s.db.GetMessagesByChains(r.Context(), chainName, "", limit)
	if err != nil {
		s.logger.Error().Err(err).Str("chain", chainName).Msg("Failed to get messages from chain")
		messagesFrom = []types.CrossChainMessage{}
	}

	messagesTo, err := s.db.GetMessagesByChains(r.Context(), "", chainName, limit)
	if err != nil {
		s.logger.Error().Err(err).Str("chain", chainName).Msg("Failed to get messages to chain")
		messagesTo = []types.CrossChainMessage{}
	}

	// Count by status
	var completedCount, failedCount int
	allMessages := append(messagesFrom, messagesTo...)
	for _, msg := range allMessages {
		switch msg.Status {
		case types.MessageStatusCompleted:
			completedCount++
		case types.MessageStatusFailed:
			failedCount++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"chain":              chainName,
		"total_messages":     len(allMessages),
		"completed_messages": completedCount,
		"failed_messages":    failedCount,
	})
}

// Transaction handlers

func (s *Server) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash := vars["hash"]

	// TODO: Query transaction from database

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tx_hash": txHash,
		"status":  "not_found",
	})
}
