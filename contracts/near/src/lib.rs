use near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};
use near_sdk::collections::{UnorderedMap, UnorderedSet};
use near_sdk::serde::{Deserialize, Serialize};
use near_sdk::{
    env, near_bindgen, AccountId, Balance, PanicOnDefault, Promise, PublicKey,
    require, log,
};

pub mod storage;
pub mod events;
pub mod types;

use storage::*;
use events::*;
use types::*;

#[near_bindgen]
#[derive(BorshDeserialize, BorshSerialize, PanicOnDefault)]
pub struct BridgeContract {
    /// Contract owner/admin
    pub owner: AccountId,

    /// Set of authorized validator public keys
    pub validators: UnorderedSet<PublicKey>,

    /// Number of signatures required for unlock
    pub required_signatures: u8,

    /// Whether the bridge is paused
    pub is_paused: bool,

    /// Total tokens locked (by token contract ID)
    pub total_locked: UnorderedMap<AccountId, Balance>,

    /// Total tokens unlocked (by token contract ID)
    pub total_unlocked: UnorderedMap<AccountId, Balance>,

    /// Processed message IDs to prevent replay
    pub processed_messages: UnorderedSet<MessageId>,

    /// Lock records for outgoing transfers
    pub lock_records: UnorderedMap<String, LockRecord>,

    /// Message counter
    pub message_count: u64,
}

#[near_bindgen]
impl BridgeContract {
    /// Initialize the bridge contract
    #[init]
    pub fn new(
        owner: AccountId,
        validators: Vec<PublicKey>,
        required_signatures: u8,
    ) -> Self {
        require!(!env::state_exists(), "Already initialized");
        require!(
            validators.len() > 0 && validators.len() <= MAX_VALIDATORS,
            "Invalid validator count"
        );
        require!(
            required_signatures > 0 && required_signatures as usize <= validators.len(),
            "Invalid required signatures"
        );

        let mut validator_set = UnorderedSet::new(StorageKey::Validators);
        for validator in validators.iter() {
            validator_set.insert(validator);
        }

        let contract = Self {
            owner,
            validators: validator_set,
            required_signatures,
            is_paused: false,
            total_locked: UnorderedMap::new(StorageKey::TotalLocked),
            total_unlocked: UnorderedMap::new(StorageKey::TotalUnlocked),
            processed_messages: UnorderedSet::new(StorageKey::ProcessedMessages),
            lock_records: UnorderedMap::new(StorageKey::LockRecords),
            message_count: 0,
        };

        log!("Bridge initialized with {} validators, requiring {} signatures",
            validators.len(),
            required_signatures
        );

        contract
    }

    /// Lock tokens for cross-chain transfer
    #[payable]
    pub fn lock_ft(
        &mut self,
        token_contract: AccountId,
        amount: U128,
        destination_chain: String,
        destination_address: String,
    ) -> Promise {
        require!(!self.is_paused, "Bridge is paused");
        require!(amount.0 > 0, "Amount must be greater than zero");
        require!(
            destination_chain.len() <= MAX_CHAIN_NAME_LEN,
            "Destination chain name too long"
        );
        require!(
            destination_address.len() <= MAX_ADDRESS_LEN,
            "Destination address too long"
        );

        let sender = env::predecessor_account_id();
        let nonce = self.message_count;
        self.message_count += 1;

        // Generate message ID
        let message_id = Self::generate_message_id(
            &sender,
            &token_contract,
            amount.0,
            &destination_chain,
            nonce,
        );

        // Create lock record
        let lock_record = LockRecord {
            message_id: message_id.clone(),
            sender: sender.clone(),
            token_contract: token_contract.clone(),
            amount: amount.0,
            destination_chain: destination_chain.clone(),
            destination_address: destination_address.clone(),
            nonce,
            timestamp: env::block_timestamp(),
        };

        self.lock_records.insert(&message_id, &lock_record);

        // Update stats
        let current_locked = self.total_locked.get(&token_contract).unwrap_or(0);
        self.total_locked.insert(&token_contract, &(current_locked + amount.0));

        // Emit event
        emit_token_locked_event(&TokenLockedEvent {
            message_id: message_id.clone(),
            sender,
            token_contract: token_contract.clone(),
            amount: amount.0,
            destination_chain,
            destination_address,
            nonce,
            timestamp: lock_record.timestamp,
        });

        log!("Tokens locked: amount={}, destination={}", amount.0, destination_chain);

        // Transfer tokens to bridge contract
        // Using ft_transfer_call to lock tokens in this contract
        ext_fungible_token::ext(token_contract)
            .with_attached_deposit(1)
            .with_static_gas(FT_TRANSFER_GAS)
            .ft_transfer(
                env::current_account_id(),
                amount,
                Some(format!("Lock for cross-chain transfer: {}", message_id)),
            )
    }

    /// Unlock tokens after cross-chain transfer (requires validator signatures)
    pub fn unlock_ft(
        &mut self,
        message_id: MessageId,
        source_chain: String,
        sender_address: String,
        recipient: AccountId,
        token_contract: AccountId,
        amount: U128,
        signatures: Vec<Signature>,
    ) -> Promise {
        require!(!self.is_paused, "Bridge is paused");
        require!(amount.0 > 0, "Amount must be greater than zero");
        require!(
            !self.processed_messages.contains(&message_id),
            "Message already processed"
        );

        // Verify signatures
        require!(
            signatures.len() >= self.required_signatures as usize,
            "Insufficient signatures"
        );

        let message_hash = Self::create_unlock_message_hash(
            &message_id,
            &source_chain,
            &sender_address,
            &recipient,
            &token_contract,
            amount.0,
        );

        let mut valid_signatures = 0;
        for sig in signatures.iter() {
            if self.verify_signature(&message_hash, sig) {
                valid_signatures += 1;
            }
        }

        require!(
            valid_signatures >= self.required_signatures as usize,
            "Insufficient valid signatures"
        );

        // Mark message as processed
        self.processed_messages.insert(&message_id);

        // Update stats
        let current_unlocked = self.total_unlocked.get(&token_contract).unwrap_or(0);
        self.total_unlocked.insert(&token_contract, &(current_unlocked + amount.0));

        // Emit event
        emit_token_unlocked_event(&TokenUnlockedEvent {
            message_id: message_id.clone(),
            source_chain,
            sender_address,
            recipient: recipient.clone(),
            token_contract: token_contract.clone(),
            amount: amount.0,
            timestamp: env::block_timestamp(),
        });

        log!("Tokens unlocked: amount={}, recipient={}", amount.0, recipient);

        // Transfer tokens from bridge to recipient
        ext_fungible_token::ext(token_contract)
            .with_attached_deposit(1)
            .with_static_gas(FT_TRANSFER_GAS)
            .ft_transfer(
                recipient,
                amount,
                Some(format!("Unlock from cross-chain transfer: {}", message_id)),
            )
    }

    /// Lock NEAR tokens for cross-chain transfer
    #[payable]
    pub fn lock_near(
        &mut self,
        destination_chain: String,
        destination_address: String,
    ) {
        require!(!self.is_paused, "Bridge is paused");
        let amount = env::attached_deposit();
        require!(amount > 0, "Amount must be greater than zero");

        let sender = env::predecessor_account_id();
        let nonce = self.message_count;
        self.message_count += 1;

        // Generate message ID
        let message_id = Self::generate_message_id(
            &sender,
            &AccountId::new_unchecked("near".to_string()),
            amount,
            &destination_chain,
            nonce,
        );

        // Create lock record
        let lock_record = LockRecord {
            message_id: message_id.clone(),
            sender: sender.clone(),
            token_contract: AccountId::new_unchecked("near".to_string()),
            amount,
            destination_chain: destination_chain.clone(),
            destination_address: destination_address.clone(),
            nonce,
            timestamp: env::block_timestamp(),
        };

        self.lock_records.insert(&message_id, &lock_record);

        // Update stats for NEAR
        let near_token = AccountId::new_unchecked("near".to_string());
        let current_locked = self.total_locked.get(&near_token).unwrap_or(0);
        self.total_locked.insert(&near_token, &(current_locked + amount));

        // Emit event
        emit_token_locked_event(&TokenLockedEvent {
            message_id: message_id.clone(),
            sender,
            token_contract: near_token,
            amount,
            destination_chain,
            destination_address,
            nonce,
            timestamp: lock_record.timestamp,
        });

        log!("NEAR locked: amount={}, destination={}", amount, destination_chain);
    }

    /// Unlock NEAR tokens after cross-chain transfer
    pub fn unlock_near(
        &mut self,
        message_id: MessageId,
        source_chain: String,
        sender_address: String,
        recipient: AccountId,
        amount: U128,
        signatures: Vec<Signature>,
    ) -> Promise {
        require!(!self.is_paused, "Bridge is paused");
        require!(amount.0 > 0, "Amount must be greater than zero");
        require!(
            !self.processed_messages.contains(&message_id),
            "Message already processed"
        );

        // Verify signatures
        require!(
            signatures.len() >= self.required_signatures as usize,
            "Insufficient signatures"
        );

        let near_token = AccountId::new_unchecked("near".to_string());
        let message_hash = Self::create_unlock_message_hash(
            &message_id,
            &source_chain,
            &sender_address,
            &recipient,
            &near_token,
            amount.0,
        );

        let mut valid_signatures = 0;
        for sig in signatures.iter() {
            if self.verify_signature(&message_hash, sig) {
                valid_signatures += 1;
            }
        }

        require!(
            valid_signatures >= self.required_signatures as usize,
            "Insufficient valid signatures"
        );

        // Mark message as processed
        self.processed_messages.insert(&message_id);

        // Update stats
        let current_unlocked = self.total_unlocked.get(&near_token).unwrap_or(0);
        self.total_unlocked.insert(&near_token, &(current_unlocked + amount.0));

        // Emit event
        emit_token_unlocked_event(&TokenUnlockedEvent {
            message_id: message_id.clone(),
            source_chain,
            sender_address,
            recipient: recipient.clone(),
            token_contract: near_token,
            amount: amount.0,
            timestamp: env::block_timestamp(),
        });

        log!("NEAR unlocked: amount={}, recipient={}", amount.0, recipient);

        // Transfer NEAR to recipient
        Promise::new(recipient).transfer(amount.0)
    }

    // ===== View methods =====

    /// Get bridge configuration
    pub fn get_config(&self) -> BridgeConfig {
        BridgeConfig {
            owner: self.owner.clone(),
            validators: self.validators.len() as u8,
            required_signatures: self.required_signatures,
            is_paused: self.is_paused,
            message_count: self.message_count,
        }
    }

    /// Check if message has been processed
    pub fn is_message_processed(&self, message_id: MessageId) -> bool {
        self.processed_messages.contains(&message_id)
    }

    /// Get lock record
    pub fn get_lock_record(&self, message_id: MessageId) -> Option<LockRecord> {
        self.lock_records.get(&message_id)
    }

    /// Get total locked for a token
    pub fn get_total_locked(&self, token_contract: AccountId) -> U128 {
        U128(self.total_locked.get(&token_contract).unwrap_or(0))
    }

    /// Get total unlocked for a token
    pub fn get_total_unlocked(&self, token_contract: AccountId) -> U128 {
        U128(self.total_unlocked.get(&token_contract).unwrap_or(0))
    }

    // ===== Admin methods =====

    /// Add a new validator
    pub fn add_validator(&mut self, validator: PublicKey) {
        self.assert_owner();
        require!(
            self.validators.len() < MAX_VALIDATORS,
            "Maximum validators reached"
        );
        require!(
            !self.validators.contains(&validator),
            "Validator already exists"
        );

        self.validators.insert(&validator);
        log!("Validator added");
    }

    /// Remove a validator
    pub fn remove_validator(&mut self, validator: PublicKey) {
        self.assert_owner();
        require!(
            self.validators.contains(&validator),
            "Validator not found"
        );

        self.validators.remove(&validator);

        // Ensure required signatures is still valid
        require!(
            self.required_signatures as usize <= self.validators.len(),
            "Invalid required signatures after removal"
        );

        log!("Validator removed");
    }

    /// Update required signatures
    pub fn update_required_signatures(&mut self, required_signatures: u8) {
        self.assert_owner();
        require!(
            required_signatures > 0 && required_signatures as usize <= self.validators.len(),
            "Invalid required signatures"
        );

        self.required_signatures = required_signatures;
        log!("Required signatures updated to: {}", required_signatures);
    }

    /// Pause the bridge
    pub fn pause(&mut self) {
        self.assert_owner();
        self.is_paused = true;
        log!("Bridge paused");
    }

    /// Unpause the bridge
    pub fn unpause(&mut self) {
        self.assert_owner();
        self.is_paused = false;
        log!("Bridge unpaused");
    }

    /// Transfer ownership
    pub fn transfer_ownership(&mut self, new_owner: AccountId) {
        self.assert_owner();
        self.owner = new_owner.clone();
        log!("Ownership transferred to: {}", new_owner);
    }

    // ===== Internal methods =====

    fn assert_owner(&self) {
        require!(
            env::predecessor_account_id() == self.owner,
            "Only owner can call this method"
        );
    }

    fn generate_message_id(
        sender: &AccountId,
        token_contract: &AccountId,
        amount: Balance,
        destination_chain: &str,
        nonce: u64,
    ) -> MessageId {
        let data = format!(
            "{}{}{}{}{}",
            sender.as_str(),
            token_contract.as_str(),
            amount,
            destination_chain,
            nonce
        );

        env::keccak256(data.as_bytes())
            .try_into()
            .expect("Hash should be 32 bytes")
    }

    fn create_unlock_message_hash(
        message_id: &MessageId,
        source_chain: &str,
        sender_address: &str,
        recipient: &AccountId,
        token_contract: &AccountId,
        amount: Balance,
    ) -> [u8; 32] {
        let mut data = Vec::new();
        data.extend_from_slice(message_id);
        data.extend_from_slice(source_chain.as_bytes());
        data.extend_from_slice(sender_address.as_bytes());
        data.extend_from_slice(recipient.as_str().as_bytes());
        data.extend_from_slice(token_contract.as_str().as_bytes());
        data.extend_from_slice(&amount.to_le_bytes());

        env::keccak256(&data)
            .try_into()
            .expect("Hash should be 32 bytes")
    }

    fn verify_signature(&self, message_hash: &[u8; 32], signature: &Signature) -> bool {
        // Verify that the signature's public key is a validator
        if !self.validators.contains(&signature.public_key) {
            return false;
        }

        // Verify Ed25519 signature
        env::ed25519_verify(
            &signature.signature,
            message_hash,
            &signature.public_key.as_bytes(),
        )
    }
}

// External contract interfaces
#[ext_contract(ext_fungible_token)]
trait FungibleToken {
    fn ft_transfer(&mut self, receiver_id: AccountId, amount: U128, memo: Option<String>);
}

// Gas constants
const FT_TRANSFER_GAS: near_sdk::Gas = near_sdk::Gas(10_000_000_000_000);

// Other constants
const MAX_VALIDATORS: usize = 10;
const MAX_CHAIN_NAME_LEN: usize = 32;
const MAX_ADDRESS_LEN: usize = 128;

// Re-exports
pub use near_sdk::json_types::U128;
