package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/trello-clone/backend/internal/config"
	"github.com/trello-clone/backend/internal/email"
	"github.com/trello-clone/backend/internal/middleware"
	"github.com/trello-clone/backend/internal/models"
	"github.com/trello-clone/backend/internal/store"
	"github.com/trello-clone/backend/internal/ws"
)

type WorkspaceHandler struct {
	store    *store.Store
	cfg      *config.Config
	hub      *ws.Hub
	mailer   *email.Mailer
	notifier *NotificationHandler
}

func NewWorkspaceHandler(s *store.Store, cfg *config.Config, hub *ws.Hub, mailer *email.Mailer, notifier *NotificationHandler) *WorkspaceHandler {
	return &WorkspaceHandler{store: s, cfg: cfg, hub: hub, mailer: mailer, notifier: notifier}
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaces, err := h.store.ListWorkspaces(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}
	if workspaces == nil {
		workspaces = []models.Workspace{}
	}
	JSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req struct {
		Name string `json:"name"`
	}
	if err := Decode(r, &req); err != nil || req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	slug := store.Slugify(req.Name)
	for i := 0; i < 10; i++ {
		trySlug := slug
		if i > 0 {
			trySlug = slug + "-" + uuid.New().String()[:8]
		}
		ws, err := h.store.CreateWorkspace(r.Context(), req.Name, trySlug, userID)
		if err == nil {
			JSON(w, http.StatusCreated, ws)
			return
		}
	}
	Error(w, http.StatusInternalServerError, "failed to create workspace")
}

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if _, err := h.store.GetMemberRole(r.Context(), id, userID); err != nil {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	ws, err := h.store.GetWorkspaceByID(r.Context(), id)
	if err != nil || ws == nil {
		Error(w, http.StatusNotFound, "workspace not found")
		return
	}
	JSON(w, http.StatusOK, ws)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	role, err := h.store.GetMemberRole(r.Context(), id, userID)
	if err != nil || !store.CanManage(role) {
		Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := Decode(r, &req); err != nil || req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	ws, err := h.store.UpdateWorkspace(r.Context(), id, req.Name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to update workspace")
		return
	}
	JSON(w, http.StatusOK, ws)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	role, err := h.store.GetMemberRole(r.Context(), id, userID)
	if err != nil || role != models.RoleOwner {
		Error(w, http.StatusForbidden, "only owner can delete workspace")
		return
	}
	if err := h.store.DeleteWorkspace(r.Context(), id); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if _, err := h.store.GetMemberRole(r.Context(), id, userID); err != nil {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	members, err := h.store.ListMembers(r.Context(), id)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	JSON(w, http.StatusOK, members)
}

func (h *WorkspaceHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	role, err := h.store.GetMemberRole(r.Context(), wsID, userID)
	if err != nil || !store.CanManage(role) {
		Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		Role models.WorkspaceRole `json:"role"`
	}
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.store.UpdateMemberRole(r.Context(), wsID, memberID, req.Role); err != nil {
		Error(w, http.StatusInternalServerError, "failed to update member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	memberID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	role, err := h.store.GetMemberRole(r.Context(), wsID, userID)
	if err != nil || !store.CanManage(role) {
		Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err := h.store.RemoveMember(r.Context(), wsID, memberID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to remove member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	role, err := h.store.GetMemberRole(r.Context(), wsID, userID)
	if err != nil || !store.CanManage(role) {
		Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		Email string               `json:"email"`
		Role  models.WorkspaceRole `json:"role"`
	}
	if err := Decode(r, &req); err != nil || req.Email == "" {
		Error(w, http.StatusBadRequest, "email is required")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if req.Role == "" {
		req.Role = models.RoleMember
	}
	if req.Role != models.RoleMember && req.Role != models.RoleAdmin {
		Error(w, http.StatusBadRequest, "role must be member or admin")
		return
	}
	if existing, _ := h.store.GetUserByEmail(r.Context(), email); existing != nil {
		if _, err := h.store.GetMemberRole(r.Context(), wsID, existing.ID); err == nil {
			Error(w, http.StatusConflict, "user is already a member")
			return
		}
	}
	if pending, _ := h.store.GetPendingInvitation(r.Context(), wsID, email); pending != nil {
		inviteURL := h.cfg.WebURL + "/invite/" + pending.Token
		JSON(w, http.StatusOK, map[string]interface{}{
			"invitation": pending,
			"invite_url": inviteURL,
		})
		return
	}
	inv, err := h.store.CreateInvitation(r.Context(), wsID, email, req.Role, userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}
	inviteURL := h.cfg.WebURL + "/invite/" + inv.Token
	ws, _ := h.store.GetWorkspaceByID(r.Context(), wsID)
	wsName := "a workspace"
	if ws != nil {
		wsName = ws.Name
	}
	body := fmt.Sprintf(
		"You've been invited to join %s.\n\nOpen this link to accept:\n\n%s\n\nThis invitation expires in 7 days.\n",
		wsName, inviteURL,
	)
	if err := h.mailer.Send(email, fmt.Sprintf("Invitation to %s", wsName), body); err != nil {
		log.Printf("invite email error: %v", err)
	}
	if existing, _ := h.store.GetUserByEmail(r.Context(), email); existing != nil && h.notifier != nil {
		inviterName := "Someone"
		if inviter, _ := h.store.GetUserByID(r.Context(), userID); inviter != nil {
			inviterName = inviter.Name
		}
		h.notifier.Push(r.Context(), existing.ID, "workspace_invite", "Workspace invitation",
			fmt.Sprintf("%s invited you to join %s", inviterName, wsName),
			"/invite/"+inv.Token)
	}
	JSON(w, http.StatusCreated, map[string]interface{}{
		"invitation": inv,
		"invite_url": inviteURL,
	})
}

func (h *WorkspaceHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	token := chi.URLParam(r, "token")
	ws, err := h.store.AcceptInvitation(r.Context(), token, userID)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, ws)
}

func RegisterWorkspaceRoutes(r chi.Router, h *WorkspaceHandler) {
	r.Get("/workspaces", h.List)
	r.Post("/workspaces", h.Create)
	r.Get("/workspaces/{id}", h.Get)
	r.Patch("/workspaces/{id}", h.Update)
	r.Delete("/workspaces/{id}", h.Delete)
	r.Get("/workspaces/{id}/members", h.ListMembers)
	r.Patch("/workspaces/{id}/members/{userId}", h.UpdateMember)
	r.Delete("/workspaces/{id}/members/{userId}", h.RemoveMember)
	r.Post("/workspaces/{id}/invitations", h.CreateInvitation)
}

func RegisterInvitationRoutes(r chi.Router, h *WorkspaceHandler) {
	r.Post("/invitations/{token}/accept", h.AcceptInvitation)
}
