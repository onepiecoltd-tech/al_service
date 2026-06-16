package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type WalletRepository interface {
	CoinPacks(ctx context.Context) ([]model.CoinPack, error)
	Topup(ctx context.Context, userID, packID uuid.UUID) (int, *model.Transaction, error)
	Gift(ctx context.Context, userID uuid.UUID, price int, description string) (int, *model.Transaction, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transaction, error)
	ListAll(ctx context.Context, limit, offset int) ([]model.AdminTransaction, int, error)
	Revenue(ctx context.Context) (*model.RevenueSummary, error)
}

type walletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) CoinPacks(ctx context.Context) ([]model.CoinPack, error) {
	rows, err := r.db.Query(ctx, `SELECT id, vnd, coins, popular FROM coin_packs ORDER BY sort`)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	packs := []model.CoinPack{}
	for rows.Next() {
		var p model.CoinPack
		if err := rows.Scan(&p.ID, &p.VND, &p.Coins, &p.Popular); err != nil {
			return nil, apperror.Internal(err)
		}
		packs = append(packs, p)
	}
	return packs, rows.Err()
}

// Topup credits the pack's coins to the user and records a transaction. Mock
// payment: always succeeds (real PayOS integration is a follow-up).
func (r *walletRepository) Topup(ctx context.Context, userID, packID uuid.UUID) (int, *model.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, nil, apperror.Internal(err)
	}
	defer tx.Rollback(ctx)

	var vnd, coins int
	err = tx.QueryRow(ctx, `SELECT vnd, coins FROM coin_packs WHERE id = $1`, packID).Scan(&vnd, &coins)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil, apperror.NotFound("coin pack not found")
		}
		return 0, nil, apperror.Internal(err)
	}

	var balance int
	if err := tx.QueryRow(ctx, `UPDATE users SET coins = coins + $2 WHERE id = $1 RETURNING coins`, userID, coins).Scan(&balance); err != nil {
		return 0, nil, apperror.Internal(err)
	}

	t := &model.Transaction{Kind: "topup", Coins: coins, VND: vnd, Method: "PayOS", Description: "Nạp xu", Status: "ok"}
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (user_id, kind, coins, vnd, method, description, status)
		 VALUES ($1,'topup',$2,$3,'PayOS','Nạp xu','ok') RETURNING id, created_at`,
		userID, coins, vnd).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return 0, nil, apperror.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, apperror.Internal(err)
	}
	return balance, t, nil
}

// Gift debits price coins (fails if balance is insufficient) and records it.
func (r *walletRepository) Gift(ctx context.Context, userID uuid.UUID, price int, description string) (int, *model.Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, nil, apperror.Internal(err)
	}
	defer tx.Rollback(ctx)

	var balance int
	if err := tx.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1`, userID).Scan(&balance); err != nil {
		return 0, nil, apperror.Internal(err)
	}
	if balance < price {
		return 0, nil, apperror.BadRequest("không đủ xu")
	}

	if err := tx.QueryRow(ctx, `UPDATE users SET coins = coins - $2 WHERE id = $1 RETURNING coins`, userID, price).Scan(&balance); err != nil {
		return 0, nil, apperror.Internal(err)
	}

	t := &model.Transaction{Kind: "gift", Coins: -price, Method: "Ví", Description: description, Status: "ok"}
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (user_id, kind, coins, method, description, status)
		 VALUES ($1,'gift',$2,'Ví',$3,'ok') RETURNING id, created_at`,
		userID, -price, description).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return 0, nil, apperror.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, apperror.Internal(err)
	}
	return balance, t, nil
}

func (r *walletRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.Transaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, kind, coins, vnd, method, description, status, created_at
		 FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	txns := []model.Transaction{}
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.Kind, &t.Coins, &t.VND, &t.Method, &t.Description, &t.Status, &t.CreatedAt); err != nil {
			return nil, apperror.Internal(err)
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *walletRepository) ListAll(ctx context.Context, limit, offset int) ([]model.AdminTransaction, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM transactions`).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT t.id, t.kind, t.coins, t.vnd, t.method, t.description, t.status, t.created_at, u.display_name
		 FROM transactions t JOIN users u ON u.id = t.user_id
		 ORDER BY t.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer rows.Close()

	txns := []model.AdminTransaction{}
	for rows.Next() {
		var t model.AdminTransaction
		if err := rows.Scan(&t.ID, &t.Kind, &t.Coins, &t.VND, &t.Method, &t.Description, &t.Status, &t.CreatedAt, &t.User); err != nil {
			return nil, 0, apperror.Internal(err)
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}

func (r *walletRepository) Revenue(ctx context.Context) (*model.RevenueSummary, error) {
	var s model.RevenueSummary
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT sum(vnd) FROM transactions WHERE status='ok' AND kind='topup' AND created_at >= date_trunc('month', now())), 0),
			COALESCE((SELECT sum(vnd) FROM transactions WHERE status='ok' AND kind='topup' AND created_at >= date_trunc('day', now())), 0),
			(SELECT count(*) FROM transactions WHERE status='ok' AND kind='topup' AND created_at >= date_trunc('month', now())),
			(SELECT count(*) FROM users WHERE plan='Pro')
	`).Scan(&s.MonthVND, &s.TodayVND, &s.TopupsMonth, &s.ProTotal)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &s, nil
}
