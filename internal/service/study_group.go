package service

import (
	"context"
	"crypto/rand"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

// Ambiguous characters (0/O, 1/I) left out so codes are easy to read aloud.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const codeLength = 6

type StudyGroupService interface {
	Create(ctx context.Context, ownerID uuid.UUID, name string) (*model.StudyGroup, error)
	// Join sends a join request by code. pending is true when the request is
	// awaiting the owner's approval (the normal case for non-owners).
	Join(ctx context.Context, userID uuid.UUID, code string) (group *model.StudyGroup, pending bool, err error)
	List(ctx context.Context, userID uuid.UUID) ([]model.StudyGroup, error)
	Members(ctx context.Context, userID, groupID uuid.UUID) ([]model.User, error)
	Leave(ctx context.Context, userID, groupID uuid.UUID) error
	// PendingRequests lists users awaiting approval — owner only.
	PendingRequests(ctx context.Context, ownerID, groupID uuid.UUID) ([]model.User, error)
	// Approve / Reject act on a pending request — owner only.
	Approve(ctx context.Context, ownerID, groupID, userID uuid.UUID) error
	Reject(ctx context.Context, ownerID, groupID, userID uuid.UUID) error
}

type studyGroupService struct {
	repo repository.StudyGroupRepository
}

func NewStudyGroupService(repo repository.StudyGroupRepository) StudyGroupService {
	return &studyGroupService{repo: repo}
}

func (s *studyGroupService) Create(ctx context.Context, ownerID uuid.UUID, name string) (*model.StudyGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperror.BadRequest("tên nhóm không được để trống")
	}
	if len(name) > 80 {
		return nil, apperror.BadRequest("tên nhóm quá dài (tối đa 80 ký tự)")
	}

	// Retry a few times on the rare code collision (unique constraint).
	var lastErr error
	for range 5 {
		g, err := s.repo.Create(ctx, name, generateCode(), ownerID)
		if err != nil {
			lastErr = err
			continue
		}
		// The owner is an approved member from the start.
		if err := s.repo.AddMember(ctx, g.ID, ownerID, "approved"); err != nil {
			return nil, err
		}
		return g, nil
	}
	return nil, lastErr
}

func (s *studyGroupService) Join(ctx context.Context, userID uuid.UUID, code string) (*model.StudyGroup, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, false, apperror.BadRequest("thiếu mã nhóm")
	}
	g, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, false, err
	}
	switch status, err := s.repo.MembershipStatus(ctx, g.ID, userID); {
	case err != nil:
		return nil, false, err
	case status == "approved":
		return nil, false, apperror.BadRequest("bạn đã ở trong nhóm này")
	case status == "pending":
		return nil, false, apperror.BadRequest("bạn đã gửi yêu cầu, đang chờ duyệt")
	}
	if err := s.repo.AddMember(ctx, g.ID, userID, "pending"); err != nil {
		return nil, false, err
	}
	return g, true, nil
}

func (s *studyGroupService) List(ctx context.Context, userID uuid.UUID) ([]model.StudyGroup, error) {
	return s.repo.ListForUser(ctx, userID)
}

func (s *studyGroupService) Members(ctx context.Context, userID, groupID uuid.UUID) ([]model.User, error) {
	// Only approved members can see the roster.
	if status, err := s.repo.MembershipStatus(ctx, groupID, userID); err != nil {
		return nil, err
	} else if status != "approved" {
		return nil, apperror.Forbidden("bạn không thuộc nhóm này")
	}
	return s.repo.Members(ctx, groupID)
}

func (s *studyGroupService) Leave(ctx context.Context, userID, groupID uuid.UUID) error {
	return s.repo.RemoveMember(ctx, groupID, userID)
}

func (s *studyGroupService) PendingRequests(ctx context.Context, ownerID, groupID uuid.UUID) ([]model.User, error) {
	if err := s.requireOwner(ctx, ownerID, groupID); err != nil {
		return nil, err
	}
	return s.repo.PendingMembers(ctx, groupID)
}

func (s *studyGroupService) Approve(ctx context.Context, ownerID, groupID, userID uuid.UUID) error {
	if err := s.requireOwner(ctx, ownerID, groupID); err != nil {
		return err
	}
	return s.repo.Approve(ctx, groupID, userID)
}

func (s *studyGroupService) Reject(ctx context.Context, ownerID, groupID, userID uuid.UUID) error {
	if err := s.requireOwner(ctx, ownerID, groupID); err != nil {
		return err
	}
	return s.repo.RemoveMember(ctx, groupID, userID)
}

// requireOwner ensures the caller is the group's owner (the "leader" who
// approves join requests), else 403/404.
func (s *studyGroupService) requireOwner(ctx context.Context, userID, groupID uuid.UUID) error {
	g, err := s.repo.FindByID(ctx, groupID)
	if err != nil {
		return err
	}
	if g.OwnerID != userID {
		return apperror.Forbidden("chỉ trưởng nhóm mới có quyền này")
	}
	return nil
}

func generateCode() string {
	b := make([]byte, codeLength)
	_, _ = rand.Read(b)
	out := make([]byte, codeLength)
	for i, c := range b {
		out[i] = codeAlphabet[int(c)%len(codeAlphabet)]
	}
	return string(out)
}
