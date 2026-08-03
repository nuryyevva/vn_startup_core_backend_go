CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount INT NOT NULL,               -- положительное = credit, отрицательное = debit
    reason TEXT NOT NULL,              -- 'dev_grant' | 'choice_purchase' | 'dialog_message'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wallet_transactions_user ON wallet_transactions(user_id);
