package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type LeaderboardHandler struct {
	leaderboard service.LeaderboardService
}

func NewLeaderboardHandler(leaderboard service.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{leaderboard: leaderboard}
}

type leaderboardRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Handle string `json:"handle"`
	Elo    int    `json:"elo"`
	Wins   int    `json:"wins"`
	Me     bool   `json:"me"`
}

type leaderboardEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []leaderboardRow `json:"data"`
}

// List godoc
//
//	@Summary	Leaderboard ranked by ELO
//	@Tags		leaderboard
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	leaderboardEnvelope
//	@Failure	401	{object}	errorEnvelope
//	@Router		/api/v1/leaderboard [get]
func (h *LeaderboardHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.leaderboard.Top(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}

	me, _ := middleware.UserIDFromContext(r.Context())
	rows := make([]leaderboardRow, len(users))
	for i, u := range users {
		rows[i] = leaderboardRow{
			ID:     u.ID.String(),
			Name:   u.DisplayName,
			Handle: u.Handle,
			Elo:    u.Elo,
			Wins:   u.Wins,
			Me:     u.ID == me,
		}
	}
	httputil.OK(w, rows)
}
