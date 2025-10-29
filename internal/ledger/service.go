package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/yzy-zzzj/ledgerlite/internal/observability"
	"github.com/yzy-zzzj/ledgerlite/internal/type"
)

type Store interface {
	CreateAccount(ctx context.Context, id, currency string, initial int64) (*types.Account, error)
	GetAccount(ctx context.Context, id string) (*types.Account, error)
	TransferTx(ctx context.Context, req types.TransferRequest) (*types.TransferResult, error)
}

type Service struct {
	store Store
	log   *observability.Logger
}

func NewService(s Store, l *observability.Logger) *Service {
	return &Service{store: s, log: l}
}

func (s *Service) CreateAccount(ctx context.Context, id, currency string, initial int64) (*types.Account, error) {
	if id == "" || currency == "" {
		return nil, errors.New("invalid account params")
	}
	if initial < 0 {
		return nil, errors.New("initial balance must be >= 0")
	}
	return s.store.CreateAccount(ctx, id, currency, initial)
}

func (s *Service) GetAccount(ctx context.Context, id string) (*types.Account, error) {
	return s.store.GetAccount(ctx, id)
}

func (s *Service) Transfer(ctx context.Context, req types.TransferRequest) (*types.TransferResult, error) {
	start := time.Now()
	res, err := s.store.TransferTx(ctx, req)
	if err != nil {
		s.log.Errorf("transfer err key=%s: %v", req.IdempotencyKey, err)
		return nil, err
	}
	s.log.Infof("transfer ok key=%s amount=%d took=%s", req.IdempotencyKey, req.AmountCents, time.Since(start))
	return res, nil
}
