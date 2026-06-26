package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/trello-clone/backend/internal/middleware"
	"github.com/trello-clone/backend/internal/models"
	"github.com/trello-clone/backend/internal/store"
	"github.com/trello-clone/backend/internal/ws"
)

type NotificationHandler struct {
	store *store.Store
	hub   *ws.Hub
}

func NewNotificationHandler(s *store.Store, hub *ws.Hub) *NotificationHandler {
	return &NotificationHandler{store: s, hub: hub}
}

// Push persists a notification and pushes it to the user in real time.
// Failures are logged but never block the triggering request.
func (h *NotificationHandler) Push(ctx context.Context, userID uuid.UUID, ntype, title, body, link string) {
	n, err := h.store.CreateNotification(ctx, userID, ntype, title, body, link)
	if err != nil {
		log.Printf("create notification: %v", err)
		return
	}
	h.hub.BroadcastUser(userID, models.WSEvent{Type: "notification.created", Payload: n})
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.store.ListNotifications(r.Context(), userID, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	JSON(w, http.StatusOK, items)
}

func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	count, err := h.store.CountUnreadNotifications(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to count notifications")
		return
	}
	JSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.store.MarkNotificationRead(r.Context(), id, userID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to mark read")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if err := h.store.MarkAllNotificationsRead(r.Context(), userID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to mark all read")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterNotificationRoutes(r chi.Router, h *NotificationHandler) {
	r.Get("/notifications", h.List)
	r.Get("/notifications/unread-count", h.UnreadCount)
	r.Post("/notifications/{id}/read", h.MarkRead)
	r.Post("/notifications/read-all", h.MarkAllRead)
}
