use near_sdk::borsh::{self, BorshSerialize};

/// Storage keys for collections
#[derive(BorshSerialize)]
pub enum StorageKey {
    Validators,
    TotalLocked,
    TotalUnlocked,
    ProcessedMessages,
    LockRecords,
}
