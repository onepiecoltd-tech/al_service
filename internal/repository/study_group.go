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

type StudyGroupRepository interface {
	Create(ctx context.Context, name, code string, ownerID uuid.UUID) (*model.StudyGroup, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.StudyGroup, error)
	FindByCode(ctx context.Context, code string) (*model.StudyGroup, error)
	// AddMember adds a member with the given status ("pending" or "approved").
	AddMember(ctx context.Context, groupID, userID uuid.UUID, status string) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
	// MembershipStatus returns "approved", "pending", or "" if no row exists.
	MembershipStatus(ctx context.Context, groupID, userID uuid.UUID) (string, error)
	// Approve flips a pending membership to approved.
	Approve(ctx context.Context, groupID, userID uuid.UUID) error
	// ListForUser returns the groups the user is an approved member of.
	ListForUser(ctx context.Context, userID uuid.UUID) ([]model.StudyGroup, error)
	// Members returns the approved members of a group.
	Members(ctx context.Context, groupID uuid.UUID) ([]model.User, error)
	// PendingMembers returns users with a pending join request for a group.
	PendingMembers(ctx context.Context, groupID uuid.UUID) ([]model.User, error)
}

type studyGroupRepository struct {
	db *pgxpool.Pool
}

func NewStudyGroupRepository(db *pgxpool.Pool) StudyGroupRepository {
	return &studyGroupRepository{db: db}
}

func (r *studyGroupRepository) Create(ctx context.Context, name, code string, ownerID uuid.UUID) (*model.StudyGroup, error) {
	var g model.StudyGroup
	err := r.db.QueryRow(ctx,
		`INSERT INTO study_groups (name, code, owner_id) VALUES ($1, $2, $3)
		 RETURNING id, name, code, owner_id, created_at`,
		name, code, ownerID).
		Scan(&g.ID, &g.Name, &g.Code, &g.OwnerID, &g.CreatedAt)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	g.MemberCount = 1
	return &g, nil
}

// memberCountSub counts approved members of g (correlated subquery).
const memberCountSub = `(SELECT count(*) FROM study_group_members m WHERE m.group_id = g.id AND m.status = 'approved')`

func (r *studyGroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.StudyGroup, error) {
	var g model.StudyGroup
	err := r.db.QueryRow(ctx,
		`SELECT g.id, g.name, g.code, g.owner_id, g.created_at, `+memberCountSub+`
		 FROM study_groups g WHERE g.id = $1`, id).
		Scan(&g.ID, &g.Name, &g.Code, &g.OwnerID, &g.CreatedAt, &g.MemberCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("không tìm thấy nhóm")
		}
		return nil, apperror.Internal(err)
	}
	return &g, nil
}

func (r *studyGroupRepository) FindByCode(ctx context.Context, code string) (*model.StudyGroup, error) {
	var g model.StudyGroup
	err := r.db.QueryRow(ctx,
		`SELECT g.id, g.name, g.code, g.owner_id, g.created_at, `+memberCountSub+`
		 FROM study_groups g WHERE g.code = $1`, code).
		Scan(&g.ID, &g.Name, &g.Code, &g.OwnerID, &g.CreatedAt, &g.MemberCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("không tìm thấy nhóm với mã này")
		}
		return nil, apperror.Internal(err)
	}
	return &g, nil
}

func (r *studyGroupRepository) AddMember(ctx context.Context, groupID, userID uuid.UUID, status string) error {
	if _, err := r.db.Exec(ctx,
		`INSERT INTO study_group_members (group_id, user_id, status) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		groupID, userID, status); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *studyGroupRepository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	if _, err := r.db.Exec(ctx,
		`DELETE FROM study_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (r *studyGroupRepository) MembershipStatus(ctx context.Context, groupID, userID uuid.UUID) (string, error) {
	var status string
	err := r.db.QueryRow(ctx,
		`SELECT status FROM study_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", apperror.Internal(err)
	}
	return status, nil
}

func (r *studyGroupRepository) Approve(ctx context.Context, groupID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE study_group_members SET status = 'approved'
		 WHERE group_id = $1 AND user_id = $2 AND status = 'pending'`,
		groupID, userID)
	if err != nil {
		return apperror.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("không tìm thấy yêu cầu tham gia")
	}
	return nil
}

func (r *studyGroupRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]model.StudyGroup, error) {
	rows, err := r.db.Query(ctx,
		`SELECT g.id, g.name, g.code, g.owner_id, g.created_at, `+memberCountSub+`
		 FROM study_groups g
		 WHERE g.id IN (SELECT group_id FROM study_group_members WHERE user_id = $1 AND status = 'approved')
		 ORDER BY g.created_at DESC`, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	groups := []model.StudyGroup{}
	for rows.Next() {
		var g model.StudyGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Code, &g.OwnerID, &g.CreatedAt, &g.MemberCount); err != nil {
			return nil, apperror.Internal(err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.Internal(err)
	}
	return groups, nil
}

func (r *studyGroupRepository) Members(ctx context.Context, groupID uuid.UUID) ([]model.User, error) {
	return r.membersByStatus(ctx, groupID, "approved", `ORDER BY elo DESC, display_name`)
}

func (r *studyGroupRepository) PendingMembers(ctx context.Context, groupID uuid.UUID) ([]model.User, error) {
	return r.membersByStatus(ctx, groupID, "pending", `ORDER BY display_name`)
}

func (r *studyGroupRepository) membersByStatus(ctx context.Context, groupID uuid.UUID, status, orderBy string) ([]model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+` FROM users
		 WHERE id IN (SELECT user_id FROM study_group_members WHERE group_id = $1 AND status = $2)
		 `+orderBy, groupID, status)
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
