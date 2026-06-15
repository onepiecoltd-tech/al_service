package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	TopByElo(ctx context.Context, limit int) ([]model.User, error)
	ListFriends(ctx context.Context, userID uuid.UUID) ([]model.User, error)
	ListAll(ctx context.Context) ([]model.User, error)
	Insert(ctx context.Context, u *model.User) error
	UpdateAdminFields(ctx context.Context, id uuid.UUID, plan, role, status string) (*model.User, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

const userColumns = `id, email, display_name, password_hash, handle, plan, coins, elo, streak, wins, presence, status_msg, role, status, created_at`

func scanUserInto(row pgx.Row, u *model.User) error {
	return row.Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash,
		&u.Handle, &u.Plan, &u.Coins, &u.Elo, &u.Streak, &u.Wins,
		&u.Presence, &u.StatusMsg, &u.Role, &u.Status, &u.CreatedAt,
	)
}

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	if err := scanUserInto(row, &u); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.Internal(err)
	}
	return &u, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *userRepository) TopByElo(ctx context.Context, limit int) ([]model.User, error) {
	rows, err := r.db.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY elo DESC, wins DESC LIMIT $1`, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := scanUserInto(rows, &u); err != nil {
			return nil, apperror.Internal(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return users, nil
}

func (r *userRepository) ListAll(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY created_at`)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := scanUserInto(rows, &u); err != nil {
			return nil, apperror.Internal(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return users, nil
}

func (r *userRepository) Insert(ctx context.Context, u *model.User) error {
	const query = `
		INSERT INTO users (email, display_name, password_hash, handle, plan, role, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	err := r.db.QueryRow(ctx, query,
		u.Email, u.DisplayName, u.PasswordHash, u.Handle, u.Plan, u.Role, u.Status,
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperror.Conflict("email already exists")
		}
		return apperror.Internal(err)
	}
	return nil
}

func (r *userRepository) UpdateAdminFields(ctx context.Context, id uuid.UUID, plan, role, status string) (*model.User, error) {
	const query = `
		UPDATE users SET plan = $2, role = $3, status = $4
		WHERE id = $1
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(ctx, query, id, plan, role, status))
}

func (r *userRepository) ListFriends(ctx context.Context, userID uuid.UUID) ([]model.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE id IN (SELECT friend_id FROM friendships WHERE user_id = $1)
		ORDER BY (presence = 'offline'), display_name`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	friends := []model.User{}
	for rows.Next() {
		var u model.User
		if err := scanUserInto(rows, &u); err != nil {
			return nil, apperror.Internal(err)
		}
		friends = append(friends, u)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return friends, nil
}
