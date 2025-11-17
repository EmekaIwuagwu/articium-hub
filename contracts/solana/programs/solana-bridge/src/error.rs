use anchor_lang::prelude::*;

#[error_code]
pub enum BridgeError {
    #[msg("Bridge is paused")]
    BridgePaused,

    #[msg("Insufficient signatures provided")]
    InsufficientSignatures,

    #[msg("Invalid signature")]
    InvalidSignature,

    #[msg("Validator not authorized")]
    UnauthorizedValidator,

    #[msg("Maximum validators reached")]
    MaxValidatorsReached,

    #[msg("Validator already exists")]
    ValidatorAlreadyExists,

    #[msg("Validator not found")]
    ValidatorNotFound,

    #[msg("Message already processed")]
    MessageAlreadyProcessed,

    #[msg("Invalid message ID")]
    InvalidMessageId,

    #[msg("Amount must be greater than zero")]
    InvalidAmount,

    #[msg("Destination chain name too long")]
    DestinationChainTooLong,

    #[msg("Destination address too long")]
    DestinationAddressTooLong,

    #[msg("Source chain name too long")]
    SourceChainTooLong,

    #[msg("Sender address too long")]
    SenderAddressTooLong,

    #[msg("Invalid required signatures count")]
    InvalidRequiredSignatures,

    #[msg("Arithmetic overflow")]
    ArithmeticOverflow,
}
