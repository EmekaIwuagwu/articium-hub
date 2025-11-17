use anchor_lang::prelude::*;
use anchor_spl::token::{self, Token, TokenAccount, Transfer};
use solana_program::ed25519_program;
use crate::state::*;
use crate::error::*;

#[derive(Accounts)]
#[instruction(message_id: [u8; 32])]
pub struct UnlockToken<'info> {
    #[account(
        mut,
        seeds = [b"bridge_config"],
        bump = bridge_config.bump,
    )]
    pub bridge_config: Account<'info, BridgeConfig>,

    #[account(
        mut,
        seeds = [b"token_vault", token_mint.key().as_ref()],
        bump = token_vault.bump,
    )]
    pub token_vault: Account<'info, TokenVault>,

    #[account(
        init,
        payer = payer,
        space = MessageRecord::LEN,
        seeds = [b"message_record", &message_id],
        bump
    )]
    pub message_record: Account<'info, MessageRecord>,

    #[account(mut)]
    pub payer: Signer<'info>,

    /// CHECK: Recipient can be any account
    pub recipient: AccountInfo<'info>,

    #[account(
        mut,
        constraint = recipient_token_account.owner == recipient.key(),
        constraint = recipient_token_account.mint == token_mint.key(),
    )]
    pub recipient_token_account: Account<'info, TokenAccount>,

    #[account(
        mut,
        constraint = vault_token_account.owner == bridge_config.key(),
        constraint = vault_token_account.mint == token_mint.key(),
    )]
    pub vault_token_account: Account<'info, TokenAccount>,

    /// CHECK: Token mint account
    pub token_mint: AccountInfo<'info>,

    pub token_program: Program<'info, Token>,
    pub system_program: Program<'info, System>,
}

pub fn handler(
    ctx: Context<UnlockToken>,
    message_id: [u8; 32],
    amount: u64,
    signatures: Vec<[u8; 64]>,
) -> Result<()> {
    let bridge_config = &ctx.accounts.bridge_config;
    let token_vault = &mut ctx.accounts.token_vault;
    let message_record = &mut ctx.accounts.message_record;

    // Check bridge is not paused
    require!(!bridge_config.is_paused, BridgeError::BridgePaused);

    // Validate amount
    require!(amount > 0, BridgeError::InvalidAmount);

    // Check sufficient signatures
    require!(
        signatures.len() >= bridge_config.required_signatures as usize,
        BridgeError::InsufficientSignatures
    );

    // Verify signatures
    let message_hash = create_unlock_message_hash(
        &message_id,
        &ctx.accounts.recipient.key(),
        &ctx.accounts.token_mint.key(),
        amount,
    );

    let mut valid_signatures = 0;
    for signature in signatures.iter() {
        for validator in bridge_config.validators.iter() {
            if verify_ed25519_signature(&message_hash, signature, validator)? {
                valid_signatures += 1;
                break;
            }
        }
    }

    require!(
        valid_signatures >= bridge_config.required_signatures as usize,
        BridgeError::InsufficientSignatures
    );

    // Transfer tokens from vault to recipient
    let bridge_config_key = bridge_config.key();
    let seeds = &[
        b"bridge_config".as_ref(),
        &[bridge_config.bump],
    ];
    let signer = &[&seeds[..]];

    let cpi_accounts = Transfer {
        from: ctx.accounts.vault_token_account.to_account_info(),
        to: ctx.accounts.recipient_token_account.to_account_info(),
        authority: ctx.accounts.bridge_config.to_account_info(),
    };
    let cpi_program = ctx.accounts.token_program.to_account_info();
    let cpi_ctx = CpiContext::new_with_signer(cpi_program, cpi_accounts, signer);
    token::transfer(cpi_ctx, amount)?;

    // Update vault stats
    token_vault.total_locked = token_vault.total_locked
        .checked_sub(amount)
        .ok_or(BridgeError::ArithmeticOverflow)?;

    // Update bridge stats (using mutable reference)
    let bridge_config_mut = &mut ctx.accounts.bridge_config;
    bridge_config_mut.total_unlocked = bridge_config_mut.total_unlocked
        .checked_add(amount)
        .ok_or(BridgeError::ArithmeticOverflow)?;

    // Create message record
    message_record.message_id = message_id;
    message_record.source_chain = String::from("external"); // Would come from signature data
    message_record.sender = String::from("unknown"); // Would come from signature data
    message_record.recipient = ctx.accounts.recipient.key();
    message_record.token_mint = ctx.accounts.token_mint.key();
    message_record.amount = amount;
    message_record.timestamp = Clock::get()?.unix_timestamp;
    message_record.processed = true;
    message_record.bump = ctx.bumps.message_record;

    msg!("Token unlocked: amount={}, recipient={}",
        amount,
        ctx.accounts.recipient.key()
    );

    // Emit event
    emit!(TokenUnlockedEvent {
        message_id,
        recipient: ctx.accounts.recipient.key(),
        token_mint: ctx.accounts.token_mint.key(),
        amount,
        timestamp: message_record.timestamp,
    });

    Ok(())
}

// Helper function to create message hash for signing
fn create_unlock_message_hash(
    message_id: &[u8; 32],
    recipient: &Pubkey,
    token_mint: &Pubkey,
    amount: u64,
) -> [u8; 32] {
    use solana_program::keccak;

    let mut data = Vec::new();
    data.extend_from_slice(message_id);
    data.extend_from_slice(recipient.as_ref());
    data.extend_from_slice(token_mint.as_ref());
    data.extend_from_slice(&amount.to_le_bytes());

    keccak::hash(&data).to_bytes()
}

// Helper function to verify Ed25519 signature
fn verify_ed25519_signature(
    message: &[u8; 32],
    signature: &[u8; 64],
    public_key: &Pubkey,
) -> Result<bool> {
    // In production, you would use ed25519_program for verification
    // For now, we'll do a simplified check
    // Note: Actual implementation would use ed25519_program::ID and verify instruction

    // This is a placeholder - in production use proper Ed25519 verification
    // via the ed25519_program
    Ok(true) // Replace with actual verification
}

#[event]
pub struct TokenUnlockedEvent {
    pub message_id: [u8; 32],
    pub recipient: Pubkey,
    pub token_mint: Pubkey,
    pub amount: u64,
    pub timestamp: i64,
}
