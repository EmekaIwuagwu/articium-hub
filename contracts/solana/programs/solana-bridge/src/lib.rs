use anchor_lang::prelude::*;
use anchor_spl::token::{self, Token, TokenAccount, Transfer};

pub mod state;
pub mod error;
pub mod instructions;

use state::*;
use error::*;
use instructions::*;

declare_id!("BrdgE1111111111111111111111111111111111111111");

#[program]
pub mod solana_bridge {
    use super::*;

    /// Initialize the bridge with validators
    pub fn initialize(
        ctx: Context<Initialize>,
        validators: Vec<Pubkey>,
        required_signatures: u8,
    ) -> Result<()> {
        instructions::initialize::handler(ctx, validators, required_signatures)
    }

    /// Lock tokens for cross-chain transfer
    pub fn lock_token(
        ctx: Context<LockToken>,
        amount: u64,
        destination_chain: String,
        destination_address: String,
        nonce: u64,
    ) -> Result<()> {
        instructions::lock_token::handler(ctx, amount, destination_chain, destination_address, nonce)
    }

    /// Unlock tokens after cross-chain transfer
    pub fn unlock_token(
        ctx: Context<UnlockToken>,
        message_id: [u8; 32],
        amount: u64,
        signatures: Vec<[u8; 64]>,
    ) -> Result<()> {
        instructions::unlock_token::handler(ctx, message_id, amount, signatures)
    }

    /// Add a new validator (admin only)
    pub fn add_validator(
        ctx: Context<UpdateValidators>,
        validator: Pubkey,
    ) -> Result<()> {
        instructions::add_validator::handler(ctx, validator)
    }

    /// Remove a validator (admin only)
    pub fn remove_validator(
        ctx: Context<UpdateValidators>,
        validator: Pubkey,
    ) -> Result<()> {
        instructions::remove_validator::handler(ctx, validator)
    }

    /// Update required signatures (admin only)
    pub fn update_required_signatures(
        ctx: Context<UpdateConfig>,
        required_signatures: u8,
    ) -> Result<()> {
        instructions::update_required_signatures::handler(ctx, required_signatures)
    }

    /// Pause the bridge (admin only)
    pub fn pause(ctx: Context<UpdateConfig>) -> Result<()> {
        instructions::pause::handler(ctx)
    }

    /// Unpause the bridge (admin only)
    pub fn unpause(ctx: Context<UpdateConfig>) -> Result<()> {
        instructions::unpause::handler(ctx)
    }
}
