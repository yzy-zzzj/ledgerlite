package ledger

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yzy-zzzj/ledgerlite/internal/observability"
	"github.com/yzy-zzzj/ledgerlite/internal/types"
)

func TestIdempotentTransfer(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/ledgerlite?sslmode=disable"
	}
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil { t.Fatal(err) }
	defer pool.Close()

	store := NewPostgresStore(pool)
	svc := NewService(store, observability.New())

	// reset
	_, _ = pool.Exec(ctx, "TRUNCATE transfers RESTART IDENTITY CASCADE")
	_, _ = pool.Exec(ctx, "TRUNCATE accounts RESTART IDENTITY CASCADE")

	_, err = svc.CreateAccount(ctx, "A", "USD", 10000)
	if err != nil { t.Fatal(err) }
	_, err = svc.CreateAccount(ctx, "B", "USD", 0)
	if err != nil { t.Fatal(err) }

	key := "idem-123"
	req := types.TransferRequest{
		IdempotencyKey: key,
		FromAccount:    "A",
		ToAccount:      "B",
		AmountCents:    5000,
		Currency:       "USD",
	}

	// Fire the same request concurrently
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := svc.Transfer(ctx, req)
			if e != nil {
				// allow serialization retries to bubble up as nil if idempotent insert wins
				errs <- e
			} else {
				errs <- nil
			}
		}()
	}
	wg.Wait()
	close(errs)

	for e := range errs {
		if e != nil && e.Error() != "insufficient funds" {
			// Under idempotency, we should see at most one commit; others either replay or conflict
			// but service returns prior committed result. Fail if unexpected.
			t.Fatalf("unexpected error: %v", e)
		}
	}

	a, _ := svc.GetAccount(ctx, "A")
	b, _ := svc.GetAccount(ctx, "B")

	if a.BalanceCents != 5000 || b.BalanceCents != 5000 {
		t.Fatalf("bad balances A=%d B=%d", a.BalanceCents, b.BalanceCents)
	}
	_ = time.Now()
}
