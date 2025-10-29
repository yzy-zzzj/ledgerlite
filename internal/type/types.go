package types

import "time"

type Account struct {
	ID           string    `json:"id"`
	Currency     string    `json:"currency"`
	BalanceCents int64     `json:"balance_cents"`
	CreatedAt    time.Time `json:"created_at"`
}

type TransferRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	FromAccount    string `json:"from_account"`
	ToAccount      string `json:"to_account"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
}

type TransferResult struct {
	IdempotencyKey string    `json:"idempotency_key"`
	FromAccount    string    `json:"from_account"`
	ToAccount      string    `json:"to_account"`
	AmountCents    int64     `json:"amount_cents"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}
