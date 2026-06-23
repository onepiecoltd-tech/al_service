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

type DuelRepository interface {
	Create(ctx context.Context, challengerID, opponentID uuid.UUID, prompt string, score int) (*model.Duel, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Duel, error)
	// Resolve records the opponent's score and outcome, and applies the ELO
	// deltas to both players, all in one transaction.
	Resolve(ctx context.Context, d *model.Duel) error
	// Decline marks a pending duel declined.
	Decline(ctx context.Context, id uuid.UUID) error
	// ListForUser returns duels the user is in (both directions), newest first.
	ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.Duel, error)
}

type duelRepository struct {
	db *pgxpool.Pool
}

func NewDuelRepository(db *pgxpool.Pool) DuelRepository {
	return &duelRepository{db: db}
}

const duelColumns = `d.id, d.challenger_id, d.opponent_id, d.prompt, d.challenger_score,
	d.opponent_score, d.status, d.winner_id, d.challenger_delta, d.opponent_delta,
	d.created_at, d.completed_at,
	c.display_name AS challenger_name, o.display_name AS opponent_name`

const duelJoins = `FROM duels d
	JOIN users c ON c.id = d.challenger_id
	JOIN users o ON o.id = d.opponent_id`

func scanDuel(row pgx.Row) (*model.Duel, error) {
	var d model.Duel
	err := row.Scan(
		&d.ID, &d.ChallengerID, &d.OpponentID, &d.Prompt, &d.ChallengerScore,
		&d.OpponentScore, &d.Status, &d.WinnerID, &d.ChallengerDelta, &d.OpponentDelta,
		&d.CreatedAt, &d.CompletedAt, &d.ChallengerName, &d.OpponentName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("không tìm thấy trận đấu")
		}
		return nil, apperror.Internal(err)
	}
	return &d, nil
}

func (r *duelRepository) Create(ctx context.Context, challengerID, opponentID uuid.UUID, prompt string, score int) (*model.Duel, error) {
	var id uuid.UUID
	if err := r.db.QueryRow(ctx,
		`INSERT INTO duels (challenger_id, opponent_id, prompt, challenger_score)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		challengerID, opponentID, prompt, score).Scan(&id); err != nil {
		return nil, apperror.Internal(err)
	}
	return r.Get(ctx, id)
}

func (r *duelRepository) Get(ctx context.Context, id uuid.UUID) (*model.Duel, error) {
	return scanDuel(r.db.QueryRow(ctx, `SELECT `+duelColumns+` `+duelJoins+` WHERE d.id = $1`, id))
}

func (r *duelRepository) Resolve(ctx context.Context, d *model.Duel) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperror.Internal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx,
		`UPDATE duels SET opponent_score = $2, status = 'completed', winner_id = $3,
		     challenger_delta = $4, opponent_delta = $5, completed_at = now()
		 WHERE id = $1 AND status = 'pending'`,
		d.ID, d.OpponentScore, d.WinnerID, d.ChallengerDelta, d.OpponentDelta); err != nil {
		return apperror.Internal(err)
	}

	// Apply ELO deltas; the winner also gets a win credited.
	if err := applyDuelElo(ctx, tx, d.ChallengerID, d.ChallengerDelta, d.WinnerID != nil && *d.WinnerID == d.ChallengerID); err != nil {
		return err
	}
	if err := applyDuelElo(ctx, tx, d.OpponentID, d.OpponentDelta, d.WinnerID != nil && *d.WinnerID == d.OpponentID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func applyDuelElo(ctx context.Context, tx pgx.Tx, userID uuid.UUID, delta int, won bool) error {
	win := 0
	if won {
		win = 1
	}
	if _, err := tx.Exec(ctx,
		`UPDATE users SET elo = GREATEST(0, elo + $2), wins = wins + $3 WHERE id = $1`,
		userID, delta, win); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *duelRepository) Decline(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `UPDATE duels SET status = 'declined' WHERE id = $1 AND status = 'pending'`, id); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *duelRepository) ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.Duel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+duelColumns+` `+duelJoins+`
		 WHERE d.challenger_id = $1 OR d.opponent_id = $1
		 ORDER BY d.created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	duels := []model.Duel{}
	for rows.Next() {
		var d model.Duel
		if err := rows.Scan(
			&d.ID, &d.ChallengerID, &d.OpponentID, &d.Prompt, &d.ChallengerScore,
			&d.OpponentScore, &d.Status, &d.WinnerID, &d.ChallengerDelta, &d.OpponentDelta,
			&d.CreatedAt, &d.CompletedAt, &d.ChallengerName, &d.OpponentName,
		); err != nil {
			return nil, apperror.Internal(err)
		}
		duels = append(duels, d)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return duels, nil
}
