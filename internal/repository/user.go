package repository

import (
	"context"
	"encoding/json"
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
	AreFriends(ctx context.Context, userA, userB uuid.UUID) (bool, error)
	// Touch records that the user was just seen (heartbeat), for presence.
	Touch(ctx context.Context, userID uuid.UUID) error
	ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.User, error)
	SearchNonFriends(ctx context.Context, userID uuid.UUID, q string, limit int) ([]model.UserSearchResult, error)
	AddFriend(ctx context.Context, userID, friendID uuid.UUID) error
	AcceptFriend(ctx context.Context, userID, requesterID uuid.UUID) error
	RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error
	GetPrefs(ctx context.Context, userID uuid.UUID) (map[string]bool, error)
	SetPrefs(ctx context.Context, userID uuid.UUID, prefs map[string]bool) error
	GetLearningLanguage(ctx context.Context, userID uuid.UUID) (string, error)
	SetLearningLanguage(ctx context.Context, userID uuid.UUID, lang string) error
	ListAll(ctx context.Context, q, plan string, limit, offset int) ([]model.User, int, error)
	Insert(ctx context.Context, u *model.User) error
	UpdateAdminFields(ctx context.Context, id uuid.UUID, plan, role, status string) (*model.User, error)
	UpdateDisplayName(ctx context.Context, id uuid.UUID, name string) (*model.User, error)
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

func (r *userRepository) UpdateDisplayName(ctx context.Context, id uuid.UUID, name string) (*model.User, error) {
	const query = `
		UPDATE users SET display_name = $2
		WHERE id = $1
		RETURNING ` + userColumns
	return scanUser(r.db.QueryRow(ctx, query, id, name))
}

// presenceExpr derives live online/offline status from last_seen_at instead
// of the static presence column: a user is "online" only if they sent a
// heartbeat within the window AND haven't hidden their status via the
// show_online privacy pref. The window is wider than the client heartbeat
// interval so one missed beat doesn't flip them offline.
const presenceExpr = `CASE
	WHEN (prefs->>'show_online') IS DISTINCT FROM 'false'
	     AND last_seen_at IS NOT NULL
	     AND last_seen_at > now() - interval '75 seconds'
	THEN 'online' ELSE 'offline' END`

// friendColumns is userColumns with the static presence column swapped for
// the computed presenceExpr — same order, so scanUserInto still applies.
const friendColumns = `id, email, display_name, password_hash, handle, plan, coins, elo, streak, wins, ` + presenceExpr + ` AS presence, status_msg, role, status, created_at`

func (r *userRepository) Touch(ctx context.Context, userID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `UPDATE users SET last_seen_at = now() WHERE id = $1`, userID); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *userRepository) ListFriends(ctx context.Context, userID uuid.UUID) ([]model.User, error) {
	const query = `
		SELECT ` + friendColumns + `
		FROM users
		WHERE id IN (SELECT friend_id FROM friendships WHERE user_id = $1 AND status = 'accepted')
		ORDER BY (` + presenceExpr + ` = 'offline'), display_name`

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

func (r *userRepository) AreFriends(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM friendships WHERE user_id = $1 AND friend_id = $2 AND status = 'accepted')`,
		userA, userB).Scan(&ok)
	if err != nil {
		return false, apperror.Internal(err)
	}
	return ok, nil
}

// SearchNonFriends excludes accepted friends and anyone who's sent userID a
// pending request (they surface via List/ListIncomingRequests instead).
// Someone userID has already sent a pending request to is still included,
// flagged via FriendStatus = "pending_sent", so the UI can grey out the
// "add friend" action instead of letting them send a duplicate request.
func (r *userRepository) SearchNonFriends(ctx context.Context, userID uuid.UUID, q string, limit int) ([]model.UserSearchResult, error) {
	const query = `
		SELECT ` + userColumns + `,
			CASE WHEN EXISTS(
				SELECT 1 FROM friendships WHERE user_id = $1 AND friend_id = users.id AND status = 'pending'
			) THEN 'pending_sent' ELSE 'none' END AS friend_status
		FROM users
		WHERE id <> $1
		  AND id NOT IN (SELECT friend_id FROM friendships WHERE user_id = $1 AND status = 'accepted')
		  AND id NOT IN (SELECT user_id FROM friendships WHERE friend_id = $1 AND status = 'pending')
		  AND ($2 = '' OR display_name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%' OR handle ILIKE '%' || $2 || '%')
		ORDER BY display_name
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, userID, q, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	users := []model.UserSearchResult{}
	for rows.Next() {
		var u model.UserSearchResult
		if err := rows.Scan(
			&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash,
			&u.Handle, &u.Plan, &u.Coins, &u.Elo, &u.Streak, &u.Wins,
			&u.Presence, &u.StatusMsg, &u.Role, &u.Status, &u.CreatedAt,
			&u.FriendStatus,
		); err != nil {
			return nil, apperror.Internal(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return users, nil
}

// AddFriend sends a friend request from userID to friendID. If friendID had
// already sent userID a pending request, this instead accepts it outright
// (both sides wanted to connect, so there's nothing left to confirm).
func (r *userRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperror.Internal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var reversePending bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM friendships WHERE user_id = $1 AND friend_id = $2 AND status = 'pending')`,
		friendID, userID).Scan(&reversePending)
	if err != nil {
		return apperror.Internal(err)
	}

	if reversePending {
		if _, err := tx.Exec(ctx,
			`UPDATE friendships SET status = 'accepted' WHERE user_id = $1 AND friend_id = $2`,
			friendID, userID); err != nil {
			return apperror.Internal(err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO friendships (user_id, friend_id, status) VALUES ($1, $2, 'accepted')
			 ON CONFLICT (user_id, friend_id) DO UPDATE SET status = 'accepted'`,
			userID, friendID); err != nil {
			return apperror.Internal(err)
		}
	} else {
		if _, err := tx.Exec(ctx,
			`INSERT INTO friendships (user_id, friend_id, status) VALUES ($1, $2, 'pending') ON CONFLICT DO NOTHING`,
			userID, friendID); err != nil {
			return apperror.Internal(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// AcceptFriend accepts a pending request that requesterID sent to userID.
func (r *userRepository) AcceptFriend(ctx context.Context, userID, requesterID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperror.Internal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	tag, err := tx.Exec(ctx,
		`UPDATE friendships SET status = 'accepted' WHERE user_id = $1 AND friend_id = $2 AND status = 'pending'`,
		requesterID, userID)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("không tìm thấy lời mời kết bạn")
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO friendships (user_id, friend_id, status) VALUES ($1, $2, 'accepted')
		 ON CONFLICT (user_id, friend_id) DO UPDATE SET status = 'accepted'`,
		userID, requesterID); err != nil {
		return apperror.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// ListIncomingRequests returns users who've sent userID a pending friend request.
func (r *userRepository) ListIncomingRequests(ctx context.Context, userID uuid.UUID) ([]model.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE id IN (SELECT user_id FROM friendships WHERE friend_id = $1 AND status = 'pending')
		ORDER BY display_name`
	rows, err := r.db.Query(ctx, query, userID)
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

func (r *userRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM friendships WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`,
		userID, friendID)
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *userRepository) GetPrefs(ctx context.Context, userID uuid.UUID) (map[string]bool, error) {
	var raw []byte
	if err := r.db.QueryRow(ctx, `SELECT prefs FROM users WHERE id = $1`, userID).Scan(&raw); err != nil {
		return nil, apperror.Internal(err)
	}
	prefs := map[string]bool{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &prefs); err != nil {
			return nil, apperror.Internal(err)
		}
	}
	return prefs, nil
}

func (r *userRepository) SetPrefs(ctx context.Context, userID uuid.UUID, prefs map[string]bool) error {
	raw, err := json.Marshal(prefs)
	if err != nil {
		return apperror.Internal(err)
	}
	if _, err := r.db.Exec(ctx, `UPDATE users SET prefs = $2 WHERE id = $1`, userID, raw); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *userRepository) GetLearningLanguage(ctx context.Context, userID uuid.UUID) (string, error) {
	var lang string
	if err := r.db.QueryRow(ctx, `SELECT learning_language FROM users WHERE id = $1`, userID).Scan(&lang); err != nil {
		return "", apperror.Internal(err)
	}
	return lang, nil
}

func (r *userRepository) SetLearningLanguage(ctx context.Context, userID uuid.UUID, lang string) error {
	if _, err := r.db.Exec(ctx, `UPDATE users SET learning_language = $2 WHERE id = $1`, userID, lang); err != nil {
		return apperror.Internal(err)
	}
	return nil
}
