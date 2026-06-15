package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type BlogHandler struct {
	blog     service.BlogService
	profiles service.ProfileService
}

func NewBlogHandler(blog service.BlogService, profiles service.ProfileService) *BlogHandler {
	return &BlogHandler{blog: blog, profiles: profiles}
}

type blogRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Excerpt  string `json:"excerpt"`
	Body     string `json:"body"`
	Status   string `json:"status" example:"published"`
}

type blogListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.BlogPost `json:"data"`
}

type blogEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data model.BlogPost `json:"data"`
}

// List godoc
//
//	@Summary	List blog posts
//	@Tags		blog
//	@Produce	json
//	@Success	200	{object}	blogListEnvelope
//	@Router		/api/v1/blog [get]
func (h *BlogHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.blog.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, posts)
}

// Get godoc
//
//	@Summary	Get a single blog post
//	@Tags		blog
//	@Produce	json
//	@Param		id	path		string	true	"Post ID"
//	@Success	200	{object}	blogEnvelope
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/blog/{id} [get]
func (h *BlogHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid post id"))
		return
	}
	post, err := h.blog.Get(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, post)
}

// Create godoc
//
//	@Summary	Create a blog post
//	@Tags		blog
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		post	body		blogRequest	true	"Post"
//	@Success	201		{object}	blogEnvelope
//	@Failure	400		{object}	errorEnvelope
//	@Failure	401		{object}	errorEnvelope
//	@Router		/api/v1/blog [post]
func (h *BlogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req blogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	post := &model.BlogPost{
		Title:    req.Title,
		Category: req.Category,
		Author:   h.authorName(r),
		Excerpt:  req.Excerpt,
		Body:     req.Body,
		Status:   req.Status,
	}
	if err := h.blog.Create(r.Context(), post); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, post)
}

// Update godoc
//
//	@Summary	Update a blog post
//	@Tags		blog
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string		true	"Post ID"
//	@Param		post	body		blogRequest	true	"Post"
//	@Success	200		{object}	blogEnvelope
//	@Failure	400		{object}	errorEnvelope
//	@Failure	401		{object}	errorEnvelope
//	@Failure	404		{object}	errorEnvelope
//	@Router		/api/v1/blog/{id} [put]
func (h *BlogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid post id"))
		return
	}
	var req blogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}

	post := &model.BlogPost{
		ID:       id,
		Title:    req.Title,
		Category: req.Category,
		Excerpt:  req.Excerpt,
		Body:     req.Body,
		Status:   req.Status,
	}
	if err := h.blog.Update(r.Context(), post); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, post)
}

// Delete godoc
//
//	@Summary	Delete a blog post
//	@Tags		blog
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path	string	true	"Post ID"
//	@Success	204	"deleted"
//	@Failure	401	{object}	errorEnvelope
//	@Failure	404	{object}	errorEnvelope
//	@Router		/api/v1/blog/{id} [delete]
func (h *BlogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid post id"))
		return
	}
	if err := h.blog.Delete(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authorName resolves the display name of the authenticated author.
func (h *BlogHandler) authorName(r *http.Request) string {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return "Ẩn danh"
	}
	if u, err := h.profiles.Get(r.Context(), id); err == nil {
		return u.DisplayName
	}
	return "Ẩn danh"
}
