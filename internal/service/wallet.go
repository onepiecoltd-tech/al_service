package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const walletHistoryLimit = 50

type TopupResult struct {
	Coins       int                `json:"coins"`
	Transaction *model.Transaction `json:"transaction"`
}

type WalletService interface {
	CoinPacks(ctx context.Context) ([]model.CoinPack, error)
	Transactions(ctx context.Context, userID uuid.UUID) ([]model.Transaction, error)
	Topup(ctx context.Context, userID, packID uuid.UUID) (*TopupResult, error)
	Gift(ctx context.Context, userID, giftID uuid.UUID) (*TopupResult, error)
	AllTransactions(ctx context.Context, limit, offset int) ([]model.AdminTransaction, int, error)
	Revenue(ctx context.Context) (*model.RevenueSummary, error)
	CreateCoinPack(ctx context.Context, vnd, coins int, popular bool) (*model.CoinPack, error)
	UpdateCoinPack(ctx context.Context, id uuid.UUID, vnd, coins int, popular bool) (*model.CoinPack, error)
	DeleteCoinPack(ctx context.Context, id uuid.UUID) error
}

type walletService struct {
	wallet repository.WalletRepository
	gifts  repository.GiftRepository
}

func NewWalletService(wallet repository.WalletRepository, gifts repository.GiftRepository) WalletService {
	return &walletService{wallet: wallet, gifts: gifts}
}

func (s *walletService) CoinPacks(ctx context.Context) ([]model.CoinPack, error) {
	return s.wallet.CoinPacks(ctx)
}

func (s *walletService) Transactions(ctx context.Context, userID uuid.UUID) ([]model.Transaction, error) {
	return s.wallet.ListByUser(ctx, userID, walletHistoryLimit)
}

func (s *walletService) Topup(ctx context.Context, userID, packID uuid.UUID) (*TopupResult, error) {
	coins, t, err := s.wallet.Topup(ctx, userID, packID)
	if err != nil {
		return nil, err
	}
	return &TopupResult{Coins: coins, Transaction: t}, nil
}

func (s *walletService) Gift(ctx context.Context, userID, giftID uuid.UUID) (*TopupResult, error) {
	gift, err := s.gifts.Get(ctx, giftID)
	if err != nil {
		return nil, err
	}
	coins, t, err := s.wallet.Gift(ctx, userID, gift.Price, fmt.Sprintf("Tặng quà %s %s", gift.Emoji, gift.Name))
	if err != nil {
		return nil, err
	}
	return &TopupResult{Coins: coins, Transaction: t}, nil
}

func (s *walletService) AllTransactions(ctx context.Context, limit, offset int) ([]model.AdminTransaction, int, error) {
	return s.wallet.ListAll(ctx, limit, offset)
}

func (s *walletService) Revenue(ctx context.Context) (*model.RevenueSummary, error) {
	return s.wallet.Revenue(ctx)
}

func (s *walletService) CreateCoinPack(ctx context.Context, vnd, coins int, popular bool) (*model.CoinPack, error) {
	if vnd <= 0 || coins <= 0 {
		return nil, apperror.BadRequest("vnd và coins phải lớn hơn 0")
	}
	p := &model.CoinPack{VND: vnd, Coins: coins, Popular: popular}
	if err := s.wallet.CreateCoinPack(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *walletService) UpdateCoinPack(ctx context.Context, id uuid.UUID, vnd, coins int, popular bool) (*model.CoinPack, error) {
	if vnd <= 0 || coins <= 0 {
		return nil, apperror.BadRequest("vnd và coins phải lớn hơn 0")
	}
	return s.wallet.UpdateCoinPack(ctx, &model.CoinPack{ID: id, VND: vnd, Coins: coins, Popular: popular})
}

func (s *walletService) DeleteCoinPack(ctx context.Context, id uuid.UUID) error {
	return s.wallet.DeleteCoinPack(ctx, id)
}
