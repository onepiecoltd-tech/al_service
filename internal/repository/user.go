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
	SearchNonFriends(ctx context.Context, userID uuid.UUID, q string, limit int) ([]model.User, error)
	AddFriend(ctx context.Context, userID, friendID uuid.UUID) error
	RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error
	ListAll(ctx context.Context, q, plan string, limit, offset int) ([]model.User, int, error)
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

func (r *userRepository) ListAll(ctx context.Context, q, plan string, limit, offset int) ([]model.User, int, error) {
	const filter = `
		WHERE ($1 = '' OR display_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR lower(plan) = $2)`

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM users`+filter, q, plan).Scan(&total); err != nil {
		return nil, 0, apperror.Internal(err)
	}

	rows, err := r.db.Query(ctx, `SELECT `+userColumns+` FROM users`+filter+` ORDER BY created_at LIMIT $3 OFFSET $4`, q, plan, limit, offset)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := scanUserInto(rows, &u); err != nil {
			return nil, 0, apperror.Internal(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.Internal(err)
	}
	return users, total, nil
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

func (r *userRepository) SearchNonFriends(ctx context.Context, userID uuid.UUID, q string, limit int) ([]model.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE id <> $1
		  AND id NOT IN (SELECT friend_id FROM friendships WHERE user_id = $1)
		  AND ($2 = '' OR display_name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%' OR handle ILIKE '%' || $2 || '%')
		ORDER BY display_name
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, userID, q, limit)
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

func (r *userRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO friendships (user_id, friend_id) VALUES ($1, $2), ($2, $1) ON CONFLICT DO NOTHING`,
		userID, friendID)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *userRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM friendships WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`,
		userID, friendID)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}
