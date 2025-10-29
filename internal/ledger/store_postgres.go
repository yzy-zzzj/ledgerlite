package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yzy-zzzj/ledgerlite/internal/types"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(p *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: p} }

func (s *PostgresStore) CreateAccount(ctx context.Context, id, currency string, initial int64) (*types.Account, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO accounts (id, currency, balance_cents)
		VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING
	`, id, currency, initial)
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, id)
}

func (s *PostgresStore) GetAccount(ctx context.Context, id string) (*types.Account, error) {
	var a types.Account
	err := s.pool.QueryRow(ctx, `
		SELECT id, currency, balance_cents, created_at FROM accounts WHERE id=$1
	`, id).Scan(&a.ID, &a.Currency, &a.BalanceCents, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// TransferTx applies idempotent transfer using SERIALIZABLE isolation and unique idempotency key.
func (s *PostgresStore) TransferTx(ctx context.Context, req types.TransferRequest) (*types.TransferResult, error) {
	if req.IdempotencyKey == "" || req.FromAccount == "" || req.ToAccount == "" || req.AmountCents <= 0 {
		return nil, errors.New("invalid transfer params")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil { return nil, err }
	defer func() { _ = tx.Rollback(ctx) }()

	// Check if idempotency key already used; if so, return prior result.
	var existing types.TransferResult
	err = tx.QueryRow(ctx, `
		SELECT idempotency_key, from_account, to_account, amount_cents, currency, status, created_at
		  FROM transfers WHERE idempotency_key=$1
	`, req.IdempotencyKey).Scan(
		&existing.IdempotencyKey, &existing.FromAccount, &existing.ToAccount,
		&existing.AmountCents, &existing.Currency, &existing.Status, &existing.CreatedAt,
	)
	if err == nil {
		// Replayed request; return previous outcome.
		_ = tx.Commit(ctx)
		return &existing, nil
	}

	// Lock accounts in deterministic order to avoid deadlocks
	a, b := req.FromAccount, req.ToAccount
	if a > b { a, b = b, a }

	var fromBal, toBal int64
	var fromCur, toCur string

	if err := tx.QueryRow(ctx, `SELECT currency, balance_cents FROM accounts WHERE id=$1 FOR UPDATE`, a).Scan(&fromCur, &fromBal); err != nil {
		return nil, fmt.Errorf("lock account %s: %w", a, err)
	}
	if err := tx.QueryRow(ctx, `SELECT currency, balance_cents FROM accounts WHERE id=$1 FOR UPDATE`, b).Scan(&toCur, &toBal); err != nil {
		return nil, fmt.Errorf("lock account %s: %w", b, err)
	}

	// Currency check (simple same-currency enforcement)
	if fromCur != toCur || fromCur != req.Currency {
		return nil, errors.New("currency mismatch")
	}

	// Ensure we reference the correct balances
	var fromCurrBal, toCurrBal int64
	if req.FromAccount == a {
		fromCurrBal = fromBal; toCurrBal = toBal
	} else {
		fromCurrBal = toBal; toCurrBal = fromBal
	}

	if fromCurrBal < req.AmountCents {
		return nil, errors.New("insufficient funds")
	}

	// Apply debits/credits
	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance_cents = balance_cents - $1 WHERE id=$2`, req.AmountCents, req.FromAccount); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance_cents = balance_cents + $1 WHERE id=$2`, req.AmountCents, req.ToAccount); err != nil {
		return nil, err
	}

	// Insert transfer record (unique on idempotency_key)
	var res types.TransferResult
	err = tx.QueryRow(ctx, `
		INSERT INTO transfers (idempotency_key, from_account, to_account, amount_cents, currency, status)
		VALUES ($1,$2,$3,$4,$5,'committed')
		RETURNING idempotency_key, from_account, to_account, amount_cents, currency, status, created_at
	`, req.IdempotencyKey, req.FromAccount, req.ToAccount, req.AmountCents, req.Currency).
		Scan(&res.IdempotencyKey, &res.FromAccount, &res.ToAccount, &res.AmountCents, &res.Currency, &res.Status, &res.CreatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil { return nil, err }
	return &res, nil
}
