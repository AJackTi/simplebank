ALTER TABLE transfers
DROP CONSTRAINT IF EXISTS transfers_accounts_distinct,
DROP CONSTRAINT IF EXISTS transfers_amount_positive;

ALTER TABLE accounts
DROP CONSTRAINT IF EXISTS accounts_balance_nonnegative;
