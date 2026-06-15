package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type AdminUserHandler struct {
	admin service.AdminUserService
}

func NewAdminUserHandler(admin service.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{admin: admin}
}

type adminUserRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Handle string `json:"handle"`
	Email  string `json:"email"`
	Plan   string `json:"plan"`
	Elo    int    `json:"elo"`
	Role   string `json:"role"`
	Status string `json:"status"`
	Joined string `json:"joined"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Handle   string `json:"handle"`
	Plan     string `json:"plan" example:"Free"`
	Role     string `json:"role" example:"user"`
}

type updateUserRequest struct {
	Plan   string `json:"plan" example:"Pro"`
	Role   string `json:"role" example:"mod"`
	Status string `json:"status" example:"active"`
}

type adminUserListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []adminUserRow `json:"data"`
}

type adminUserEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data adminUserRow `json:"data"`
}

func toRow(u *model.User) adminUserRow {
	return adminUserRow{
		ID:     u.ID.String(),
		Name:   u.DisplayName,
		Handle: u.Handle,
		Email:  u.Email,
		Plan:   u.Plan,
		Elo:    u.Elo,
		Role:   u.Role,
		Status: u.Status,
		Joined: u.CreatedAt.Format("01/2006"),
	}
}

// List godoc
//
//	@Summary	List all users (admin)
//	@Tags		admin
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	adminUserListEnvelope
//	@Failure	401	{object}	errorEnvelope
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/admin/users [get]
func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.admin.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	rows := make([]adminUserRow, len(users))
	for i := range users {
		rows[i] = toRow(&users[i])
	}
	httputil.OK(w, rows)
}

// Create godoc
//
//	@Summary	Create a user (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		user	body		createUserRequest	true	"User"
//	@Success	201		{object}	adminUserEnvelope
//	@Failure	400		{object}	errorEnvelope
//	@Failure	409		{object}	errorEnvelope
//	@Router		/api/v1/admin/users [post]
func (h *AdminUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	u, err := h.admin.Create(r.Context(), service.NewUserInput{
		Email:       req.Email,
		DisplayName: req.Name,
		Password:    req.Password,
		Handle:      req.Handle,
		Plan:        req.Plan,
		Role:        req.Role,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, toRow(u))
}

// Update godoc
//
//	@Summary	Update a user's plan/role/status (admin)
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string				true	"User ID"
//	@Param		user	body		updateUserRequest	true	"Fields"
//	@Success	200		{object}	adminUserEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/admin/users/{id} [put]
func (h *AdminUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid user id"))
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	u, err := h.admin.Update(r.Context(), id, req.Plan, req.Role, req.Status)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, toRow(u))
}
