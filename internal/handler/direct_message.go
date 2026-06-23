package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/craftbyte/learning_languages/services/internal/apperror"
	"github.com/craftbyte/learning_languages/services/internal/middleware"
	"github.com/craftbyte/learning_languages/services/internal/service"
	"github.com/craftbyte/learning_languages/services/pkg/httputil"
)

type DirectMessageHandler struct {
	messages  service.DirectMessageService
	jwtSecret string
	isActive  func(context.Context, uuid.UUID) (bool, error)
}

func NewDirectMessageHandler(messages service.DirectMessageService, jwtSecret string, isActive func(context.Context, uuid.UUID) (bool, error)) *DirectMessageHandler {
	return &DirectMessageHandler{messages: messages, jwtSecret: jwtSecret, isActive: isActive}
}

// History godoc
//
//	@Summary	Get the message history with a friend
//	@Tags		messages
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		string	true	"Friend user id"
//	@Success	200	{object}	map[string]interface{}
//	@Failure	403	{object}	errorEnvelope
//	@Router		/api/v1/messages/{id} [get]
func (h *DirectMessageHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	otherID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid user id"))
		return
	}
	msgs, err := h.messages.History(r.Context(), id, otherID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, msgs)
}

type sendMessageRequest struct {
	Body string `json:"body"`
}

// Send godoc
//
//	@Summary	Send a direct message to a friend
//	@Tags		messages
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		string				true	"Friend user id"
//	@Param		body	body		sendMessageRequest	true	"Message"
//	@Success	201		{object}	map[string]interface{}
//	@Failure	400		{object}	errorEnvelope
//	@Failure	403		{object}	errorEnvelope
//	@Router		/api/v1/messages/{id} [post]
func (h *DirectMessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.RequireUserID(w, r)
	if !ok {
		return
	}
	otherID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.Error(w, apperror.BadRequest("invalid user id"))
		return
	}
	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.BadRequest("invalid request body"))
		return
	}
	msg, err := h.messages.Send(r.Context(), id, otherID, req.Body)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, msg)
}

var wsUpgrader = websocket.Upgrader{
	// This endpoint is only ever called server-to-server by the Nitro BFF
	// (see web/server/routes/api/messages/ws.ts), never directly by a
	// browser, so there's no third-party origin to defend against here.
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	wsWriteWait  = 10 * time.Second
	wsPingPeriod = 25 * time.Second
	wsPongWait   = wsPingPeriod + wsWriteWait
)

// Stream godoc
//
//	@Summary	WebSocket stream of incoming direct messages
//	@Description	Upgrades to a WebSocket and pushes every message sent to the authenticated user, from any friend, as one JSON text frame per message. Authenticated via the standard Authorization header, or a "?token=" query param (needed because browser WebSocket clients can't set custom headers — used by the Nitro BFF's server-to-server connection here, never exposed to the actual browser). Message history still comes from GET /api/v1/messages/{id}.
//	@Tags		messages
//	@Router		/api/v1/messages/stream [get]
func (h *DirectMessageHandler) Stream(w http.ResponseWriter, r *http.Request) {
	id, ok := middleware.AuthenticateWS(h.jwtSecret, r)
	if !ok {
		httputil.Error(w, apperror.Unauthorized("invalid or missing token"))
		return
	}
	if active, err := h.isActive(r.Context(), id); err != nil || !active {
		httputil.Error(w, apperror.Forbidden("tài khoản đã bị khóa"))
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote an HTTP error response
	}
	defer conn.Close()

	ch, unsubscribe := h.messages.Subscribe(id)
	defer unsubscribe()

	// Read pump: the only thing we expect from the client is pong frames
	// (handled automatically by the library) and the close frame — this
	// loop's sole job is to detect that close and unblock the write pump.
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	ping := time.NewTicker(wsPingPeriod)
	defer ping.Stop()

	for {
		select {
		case <-closed:
			return
		case <-r.Context().Done():
			return
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case msg, ok := <-ch:
			if !ok {
				return
			}
			buf, err := json.Marshal(msg)
			if err != nil {
				slog.Error("marshal direct message for ws", "error", err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.TextMessage, buf); err != nil {
				return
			}
		}
	}
}
