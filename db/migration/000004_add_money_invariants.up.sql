ALTER TABLE accounts
ADD CONSTRAINT accounts_balance_nonnegative CHECK (balance >= 0);

ALTER TABLE transfers
ADD CONSTRAINT transfers_amount_positive CHECK (amount > 0),
ADD CONSTRAINT transfers_accounts_distinct CHECK (from_account_id <> to_account_id);
