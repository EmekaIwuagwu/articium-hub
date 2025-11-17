use anchor_lang::prelude::*;
use crate::state::*;
use crate::error::*;

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(
        init,
        payer = admin,
        space = BridgeConfig::LEN,
        seeds = [b"bridge_config"],
        bump
    )]
    pub bridge_config: Account<'info, BridgeConfig>,

    #[account(mut)]
    pub admin: Signer<'info>,

    pub system_program: Program<'info, System>,
}

pub fn handler(
    ctx: Context<Initialize>,
    validators: Vec<Pubkey>,
    required_signatures: u8,
) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;

    // Validate inputs
    require!(
        validators.len() <= BridgeConfig::MAX_VALIDATORS,
        BridgeError::MaxValidatorsReached
    );

    require!(
        required_signatures > 0 && required_signatures as usize <= validators.len(),
        BridgeError::InvalidRequiredSignatures
    );

    // Initialize bridge config
    bridge_config.admin = ctx.accounts.admin.key();
    bridge_config.validators = validators;
    bridge_config.required_signatures = required_signatures;
    bridge_config.is_paused = false;
    bridge_config.total_locked = 0;
    bridge_config.total_unlocked = 0;
    bridge_config.message_count = 0;
    bridge_config.bump = ctx.bumps.bridge_config;

    msg!("Bridge initialized with {} validators, requiring {} signatures",
        bridge_config.validators.len(),
        bridge_config.required_signatures
    );

    Ok(())
}
