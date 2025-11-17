use anchor_lang::prelude::*;
use anchor_spl::token::{self, Token, TokenAccount, Transfer};
use solana_program::keccak;
use crate::state::*;
use crate::error::*;

#[derive(Accounts)]
#[instruction(amount: u64, destination_chain: String, destination_address: String, nonce: u64)]
pub struct LockToken<'info> {
    #[account(
        mut,
        seeds = [b"bridge_config"],
        bump = bridge_config.bump,
    )]
    pub bridge_config: Account<'info, BridgeConfig>,

    #[account(
        init_if_needed,
        payer = sender,
        space = TokenVault::LEN,
        seeds = [b"token_vault", token_mint.key().as_ref()],
        bump
    )]
    pub token_vault: Account<'info, TokenVault>,

    #[account(
        init,
        payer = sender,
        space = LockRecord::LEN,
        seeds = [b"lock_record", sender.key().as_ref(), &nonce.to_le_bytes()],
        bump
    )]
    pub lock_record: Account<'info, LockRecord>,

    #[account(mut)]
    pub sender: Signer<'info>,

    #[account(
        mut,
        constraint = sender_token_account.owner == sender.key(),
        constraint = sender_token_account.mint == token_mint.key(),
    )]
    pub sender_token_account: Account<'info, TokenAccount>,

    #[account(
        init_if_needed,
        payer = sender,
        associated_token::mint = token_mint,
        associated_token::authority = bridge_config,
    )]
    pub vault_token_account: Account<'info, TokenAccount>,

    /// CHECK: Token mint account
    pub token_mint: AccountInfo<'info>,

    pub token_program: Program<'info, Token>,
    pub associated_token_program: Program<'info, anchor_spl::associated_token::AssociatedToken>,
    pub system_program: Program<'info, System>,
    pub rent: Sysvar<'info, Rent>,
}

pub fn handler(
    ctx: Context<LockToken>,
    amount: u64,
    destination_chain: String,
    destination_address: String,
    nonce: u64,
) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;
    let token_vault = &mut ctx.accounts.token_vault;
    let lock_record = &mut ctx.accounts.lock_record;

    // Check bridge is not paused
    require!(!bridge_config.is_paused, BridgeError::BridgePaused);

    // Validate amount
    require!(amount > 0, BridgeError::InvalidAmount);

    // Validate string lengths
    require!(
        destination_chain.len() <= LockRecord::MAX_CHAIN_LEN,
        BridgeError::DestinationChainTooLong
    );
    require!(
        destination_address.len() <= LockRecord::MAX_ADDRESS_LEN,
        BridgeError::DestinationAddressTooLong
    );

    // Generate message ID from hash of lock parameters
    let message_data = format!(
        "{}{}{}{}{}",
        ctx.accounts.sender.key(),
        ctx.accounts.token_mint.key(),
        amount,
        destination_chain,
        nonce
    );
    let message_id = keccak::hash(message_data.as_bytes()).to_bytes();

    // Transfer tokens to vault
    let cpi_accounts = Transfer {
        from: ctx.accounts.sender_token_account.to_account_info(),
        to: ctx.accounts.vault_token_account.to_account_info(),
        authority: ctx.accounts.sender.to_account_info(),
    };
    let cpi_program = ctx.accounts.token_program.to_account_info();
    let cpi_ctx = CpiContext::new(cpi_program, cpi_accounts);
    token::transfer(cpi_ctx, amount)?;

    // Initialize token vault if needed
    if token_vault.bridge_config == Pubkey::default() {
        token_vault.bridge_config = bridge_config.key();
        token_vault.token_mint = ctx.accounts.token_mint.key();
        token_vault.total_locked = 0;
        token_vault.bump = ctx.bumps.token_vault;
    }

    // Update vault stats
    token_vault.total_locked = token_vault.total_locked
        .checked_add(amount)
        .ok_or(BridgeError::ArithmeticOverflow)?;

    // Update bridge stats
    bridge_config.total_locked = bridge_config.total_locked
        .checked_add(amount)
        .ok_or(BridgeError::ArithmeticOverflow)?;
    bridge_config.message_count = bridge_config.message_count
        .checked_add(1)
        .ok_or(BridgeError::ArithmeticOverflow)?;

    // Create lock record
    lock_record.message_id = message_id;
    lock_record.sender = ctx.accounts.sender.key();
    lock_record.destination_chain = destination_chain.clone();
    lock_record.destination_address = destination_address.clone();
    lock_record.token_mint = ctx.accounts.token_mint.key();
    lock_record.amount = amount;
    lock_record.nonce = nonce;
    lock_record.timestamp = Clock::get()?.unix_timestamp;
    lock_record.bump = ctx.bumps.lock_record;

    msg!("Token locked: amount={}, destination={}, address={}",
        amount,
        destination_chain,
        destination_address
    );

    // Emit event
    emit!(TokenLockedEvent {
        message_id,
        sender: ctx.accounts.sender.key(),
        token_mint: ctx.accounts.token_mint.key(),
        amount,
        destination_chain,
        destination_address,
        nonce,
        timestamp: lock_record.timestamp,
    });

    Ok(())
}

#[event]
pub struct TokenLockedEvent {
    pub message_id: [u8; 32],
    pub sender: Pubkey,
    pub token_mint: Pubkey,
    pub amount: u64,
    pub destination_chain: String,
    pub destination_address: String,
    pub nonce: u64,
    pub timestamp: i64,
}
