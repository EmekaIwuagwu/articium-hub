use anchor_lang::prelude::*;
use crate::state::*;
use crate::error::*;

// ===== Add Validator =====

#[derive(Accounts)]
pub struct UpdateValidators<'info> {
    #[account(
        mut,
        seeds = [b"bridge_config"],
        bump = bridge_config.bump,
        constraint = bridge_config.admin == admin.key() @ BridgeError::UnauthorizedValidator
    )]
    pub bridge_config: Account<'info, BridgeConfig>,

    pub admin: Signer<'info>,
}

pub fn add_validator(ctx: Context<UpdateValidators>, validator: Pubkey) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;

    // Check if validator already exists
    require!(
        !bridge_config.is_validator(&validator),
        BridgeError::ValidatorAlreadyExists
    );

    // Check max validators
    require!(
        bridge_config.validators.len() < BridgeConfig::MAX_VALIDATORS,
        BridgeError::MaxValidatorsReached
    );

    // Add validator
    bridge_config.validators.push(validator);

    msg!("Validator added: {}", validator);

    Ok(())
}

pub fn remove_validator(ctx: Context<UpdateValidators>, validator: Pubkey) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;

    // Find and remove validator
    let position = bridge_config.validators.iter().position(|v| v == &validator);
    require!(position.is_some(), BridgeError::ValidatorNotFound);

    bridge_config.validators.remove(position.unwrap());

    // Ensure required signatures is still valid
    require!(
        bridge_config.required_signatures as usize <= bridge_config.validators.len(),
        BridgeError::InvalidRequiredSignatures
    );

    msg!("Validator removed: {}", validator);

    Ok(())
}

// ===== Update Config =====

#[derive(Accounts)]
pub struct UpdateConfig<'info> {
    #[account(
        mut,
        seeds = [b"bridge_config"],
        bump = bridge_config.bump,
        constraint = bridge_config.admin == admin.key() @ BridgeError::UnauthorizedValidator
    )]
    pub bridge_config: Account<'info, BridgeConfig>,

    pub admin: Signer<'info>,
}

pub fn update_required_signatures(
    ctx: Context<UpdateConfig>,
    required_signatures: u8,
) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;

    // Validate new required signatures
    require!(
        required_signatures > 0 && required_signatures as usize <= bridge_config.validators.len(),
        BridgeError::InvalidRequiredSignatures
    );

    bridge_config.required_signatures = required_signatures;

    msg!("Required signatures updated to: {}", required_signatures);

    Ok(())
}

pub fn pause(ctx: Context<UpdateConfig>) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;

    bridge_config.is_paused = true;

    msg!("Bridge paused");

    Ok(())
}

pub fn unpause(ctx: Context<UpdateConfig>) -> Result<()> {
    let bridge_config = &mut ctx.accounts.bridge_config;

    bridge_config.is_paused = false;

    msg!("Bridge unpaused");

    Ok(())
}
