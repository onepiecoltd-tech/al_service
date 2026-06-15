package handler

import (
	"net/http"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type NotificationHandler struct {
	notifications service.NotificationService
}

func NewNotificationHandler(notifications service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

type notificationListEnvelope struct { //nolint:unused // referenced by swaggo annotations only
	Data []model.Notification `json:"data"`
}

// List godoc
//
//	@Summary	List the authenticated user's notifications
//	@Tags		notifications
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	notificationListEnvelope
//	@Failure	401	{object}	errorEnvelope
//	@Router		/api/v1/notifications [get]
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}
	items, err := h.notifications.List(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, items)
}

// MarkAllRead godoc
//
//	@Summary	Mark all notifications as read
//	@Tags		notifications
//	@Produce	json
//	@Security	BearerAuth
//	@Success	204	"marked read"
//	@Failure	401	{object}	errorEnvelope
//	@Router		/api/v1/notifications/read [post]
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.Unauthorized("not authenticated"))
		return
	}
	if err := h.notifications.MarkAllRead(r.Context(), id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
