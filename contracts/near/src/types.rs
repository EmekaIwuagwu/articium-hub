use near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};
use near_sdk::serde::{Deserialize, Serialize};
use near_sdk::{AccountId, Balance, PublicKey};

/// Message ID type (32 bytes)
pub type MessageId = [u8; 32];

/// Lock record for outgoing cross-chain transfers
#[derive(BorshDeserialize, BorshSerialize, Serialize, Deserialize, Clone)]
#[serde(crate = "near_sdk::serde")]
pub struct LockRecord {
    pub message_id: MessageId,
    pub sender: AccountId,
    pub token_contract: AccountId,
    pub amount: Balance,
    pub destination_chain: String,
    pub destination_address: String,
    pub nonce: u64,
    pub timestamp: u64,
}

/// Signature from a validator
#[derive(BorshDeserialize, BorshSerialize, Serialize, Deserialize, Clone, Debug)]
#[serde(crate = "near_sdk::serde")]
pub struct Signature {
    pub public_key: PublicKey,
    pub signature: Vec<u8>,
}

/// Bridge configuration (view)
#[derive(Serialize, Deserialize)]
#[serde(crate = "near_sdk::serde")]
pub struct BridgeConfig {
    pub owner: AccountId,
    pub validators: u8,
    pub required_signatures: u8,
    pub is_paused: bool,
    pub message_count: u64,
}
