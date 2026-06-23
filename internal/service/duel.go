package service

import (
	"context"
	"math"
	"strings"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const (
	duelHistoryLimit = 30
	eloK             = 24 // ELO sensitivity per match
)

type DuelService interface {
	// Challenge creates a pending duel: the challenger has already recorded
	// their pronunciation score; the opponent (a friend) is invited to beat it.
	Challenge(ctx context.Context, challengerID, opponentID uuid.UUID, prompt string, score int) (*model.Duel, error)
	// Respond records the opponent's score, decides the winner, applies ELO.
	Respond(ctx context.Context, opponentID, duelID uuid.UUID, score int) (*model.Duel, error)
	// Decline rejects a pending duel (opponent only).
	Decline(ctx context.Context, opponentID, duelID uuid.UUID) error
	// List returns the user's recent duels (both directions).
	List(ctx context.Context, userID uuid.UUID) ([]model.Duel, error)
}

type duelService struct {
	duels repository.DuelRepository
	users repository.UserRepository
}

func NewDuelService(duels repository.DuelRepository, users repository.UserRepository) DuelService {
	return &duelService{duels: duels, users: users}
}

func (s *duelService) Challenge(ctx context.Context, challengerID, opponentID uuid.UUID, prompt string, score int) (*model.Duel, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, apperror.BadRequest("thiếu nội dung thử thách")
	}
	if challengerID == opponentID {
		return nil, apperror.BadRequest("không thể tự thách đấu chính mình")
	}
	score = clampScore(score)
	// Duels are between friends only.
	ok, err := s.users.AreFriends(ctx, challengerID, opponentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Forbidden("chỉ có thể thách đấu bạn bè")
	}
	return s.duels.Create(ctx, challengerID, opponentID, prompt, score)
}

func (s *duelService) Respond(ctx context.Context, opponentID, duelID uuid.UUID, score int) (*model.Duel, error) {
	d, err := s.duels.Get(ctx, duelID)
	if err != nil {
		return nil, err
	}
	if d.OpponentID != opponentID {
		return nil, apperror.Forbidden("đây không phải lời thách đấu dành cho bạn")
	}
	if d.Status != "pending" {
		return nil, apperror.BadRequest("trận đấu này đã kết thúc")
	}
	score = clampScore(score)
	d.OpponentScore = &score

	// Current ratings, for the ELO exchange.
	challenger, err := s.users.FindByID(ctx, d.ChallengerID)
	if err != nil {
		return nil, err
	}
	opponent, err := s.users.FindByID(ctx, opponentID)
	if err != nil {
		return nil, err
	}

	// Outcome from the two pronunciation scores (higher wins; tie = draw).
	var challengerOutcome float64
	switch {
	case d.ChallengerScore > score:
		challengerOutcome = 1
		d.WinnerID = &d.ChallengerID
	case d.ChallengerScore < score:
		challengerOutcome = 0
		d.WinnerID = &d.OpponentID
	default:
		challengerOutcome = 0.5 // draw
	}

	d.ChallengerDelta = eloDelta(challenger.Elo, opponent.Elo, challengerOutcome)
	d.OpponentDelta = eloDelta(opponent.Elo, challenger.Elo, 1-challengerOutcome)

	if err := s.duels.Resolve(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *duelService) Decline(ctx context.Context, opponentID, duelID uuid.UUID) error {
	d, err := s.duels.Get(ctx, duelID)
	if err != nil {
		return err
	}
	if d.OpponentID != opponentID {
		return apperror.Forbidden("đây không phải lời thách đấu dành cho bạn")
	}
	return s.duels.Decline(ctx, duelID)
}

func (s *duelService) List(ctx context.Context, userID uuid.UUID) ([]model.Duel, error) {
	return s.duels.ListForUser(ctx, userID, duelHistoryLimit)
}

func clampScore(s int) int {
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
}

// eloDelta is the standard Elo rating change for `rating` against `opponent`
// given the actual result (1 win / 0.5 draw / 0 loss).
func eloDelta(rating, opponent int, result float64) int {
	expected := 1.0 / (1.0 + math.Pow(10, float64(opponent-rating)/400.0))
	return int(math.Round(eloK * (result - expected)))
}
