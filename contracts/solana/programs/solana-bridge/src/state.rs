use anchor_lang::prelude::*;

/// Bridge configuration and state
#[account]
pub struct BridgeConfig {
    /// Authority that can manage the bridge
    pub admin: Pubkey,

    /// List of authorized validators (max 10)
    pub validators: Vec<Pubkey>,

    /// Number of signatures required for unlock
    pub required_signatures: u8,

    /// Whether the bridge is paused
    pub is_paused: bool,

    /// Total tokens locked
    pub total_locked: u64,

    /// Total tokens unlocked
    pub total_unlocked: u64,

    /// Number of messages processed
    pub message_count: u64,

    /// Bump seed for PDA
    pub bump: u8,
}

impl BridgeConfig {
    pub const MAX_VALIDATORS: usize = 10;

    pub const LEN: usize = 8 + // discriminator
        32 + // admin
        (4 + 32 * Self::MAX_VALIDATORS) + // validators vec
        1 + // required_signatures
        1 + // is_paused
        8 + // total_locked
        8 + // total_unlocked
        8 + // message_count
        1; // bump

    pub fn is_validator(&self, pubkey: &Pubkey) -> bool {
        self.validators.contains(pubkey)
    }
}

/// Token vault for holding locked tokens
#[account]
pub struct TokenVault {
    /// The bridge config this vault belongs to
    pub bridge_config: Pubkey,

    /// The token mint
    pub token_mint: Pubkey,

    /// Total amount locked in this vault
    pub total_locked: u64,

    /// Bump seed for PDA
    pub bump: u8,
}

impl TokenVault {
    pub const LEN: usize = 8 + // discriminator
        32 + // bridge_config
        32 + // token_mint
        8 + // total_locked
        1; // bump
}

/// Record of a cross-chain message
#[account]
pub struct MessageRecord {
    /// Unique message ID
    pub message_id: [u8; 32],

    /// Source chain identifier
    pub source_chain: String,

    /// Sender address on source chain
    pub sender: String,

    /// Recipient on Solana
    pub recipient: Pubkey,

    /// Token mint
    pub token_mint: Pubkey,

    /// Amount transferred
    pub amount: u64,

    /// Timestamp when locked
    pub timestamp: i64,

    /// Whether this message has been processed
    pub processed: bool,

    /// Bump seed for PDA
    pub bump: u8,
}

impl MessageRecord {
    pub const MAX_CHAIN_LEN: usize = 32;
    pub const MAX_SENDER_LEN: usize = 128;

    pub const LEN: usize = 8 + // discriminator
        32 + // message_id
        (4 + Self::MAX_CHAIN_LEN) + // source_chain
        (4 + Self::MAX_SENDER_LEN) + // sender
        32 + // recipient
        32 + // token_mint
        8 + // amount
        8 + // timestamp
        1 + // processed
        1; // bump
}

/// Record of a lock event for outgoing transfers
#[account]
pub struct LockRecord {
    /// Message ID
    pub message_id: [u8; 32],

    /// Sender on Solana
    pub sender: Pubkey,

    /// Destination chain
    pub destination_chain: String,

    /// Destination address
    pub destination_address: String,

    /// Token mint
    pub token_mint: Pubkey,

    /// Amount locked
    pub amount: u64,

    /// Nonce
    pub nonce: u64,

    /// Timestamp when locked
    pub timestamp: i64,

    /// Bump seed for PDA
    pub bump: u8,
}

impl LockRecord {
    pub const MAX_CHAIN_LEN: usize = 32;
    pub const MAX_ADDRESS_LEN: usize = 128;

    pub const LEN: usize = 8 + // discriminator
        32 + // message_id
        32 + // sender
        (4 + Self::MAX_CHAIN_LEN) + // destination_chain
        (4 + Self::MAX_ADDRESS_LEN) + // destination_address
        32 + // token_mint
        8 + // amount
        8 + // nonce
        8 + // timestamp
        1; // bump
}
