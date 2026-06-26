package handler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/trello-clone/backend/internal/middleware"
	"github.com/trello-clone/backend/internal/models"
	"github.com/trello-clone/backend/internal/store"
	"github.com/trello-clone/backend/internal/ws"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

type BoardHandler struct {
	store    *store.Store
	hub      *ws.Hub
	notifier *NotificationHandler
}

func NewBoardHandler(s *store.Store, hub *ws.Hub, notifier *NotificationHandler) *BoardHandler {
	return &BoardHandler{store: s, hub: hub, notifier: notifier}
}

func (h *BoardHandler) boardLink(ctx context.Context, boardID uuid.UUID) string {
	slug, err := h.store.GetBoardSlug(ctx, boardID)
	if err != nil || slug == "" {
		return ""
	}
	return fmt.Sprintf("/w/%s/b/%s", slug, boardID)
}

func (h *BoardHandler) requireMember(r *http.Request, workspaceID uuid.UUID) (uuid.UUID, models.WorkspaceRole, bool) {
	userID := middleware.UserIDFromContext(r.Context())
	role, err := h.store.GetMemberRole(r.Context(), workspaceID, userID)
	if err != nil {
		return userID, "", false
	}
	return userID, role, true
}

func (h *BoardHandler) ListBoards(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	boards, err := h.store.ListBoards(r.Context(), wsID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list boards")
		return
	}
	if boards == nil {
		boards = []models.Board{}
	}
	JSON(w, http.StatusOK, boards)
}

func (h *BoardHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := Decode(r, &req); err != nil || req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	board, err := h.store.CreateBoard(r.Context(), wsID, req.Name, req.Description)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create board")
		return
	}
	_, _ = h.store.CreateList(r.Context(), board.ID, "To Do")
	_, _ = h.store.CreateList(r.Context(), board.ID, "In Progress")
	_, _ = h.store.CreateList(r.Context(), board.ID, "Done")
	JSON(w, http.StatusCreated, board)
}

func (h *BoardHandler) GetBoard(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid board id")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	detail, err := h.store.GetBoardDetail(r.Context(), boardID)
	if err != nil || detail == nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	JSON(w, http.StatusOK, detail)
}

func (h *BoardHandler) UpdateBoard(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid board id")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	_, role, ok := h.requireMember(r, wsID)
	if !ok || !store.CanManage(role) {
		Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := Decode(r, &req); err != nil || req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	board, err := h.store.UpdateBoard(r.Context(), boardID, req.Name, req.Description)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to update board")
		return
	}
	JSON(w, http.StatusOK, board)
}

func (h *BoardHandler) DeleteBoard(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid board id")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	_, role, ok := h.requireMember(r, wsID)
	if !ok || !store.CanManage(role) {
		Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err := h.store.DeleteBoard(r.Context(), boardID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete board")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid board id")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := Decode(r, &req); err != nil || req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	list, err := h.store.CreateList(r.Context(), boardID, req.Name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create list")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{Type: "list.created", BoardID: boardID, Payload: list})
	JSON(w, http.StatusCreated, list)
}

func (h *BoardHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	listID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid list id")
		return
	}
	list, err := h.store.GetList(r.Context(), listID)
	if err != nil || list == nil {
		Error(w, http.StatusNotFound, "list not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), list.BoardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := Decode(r, &req); err != nil || req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	updated, err := h.store.UpdateList(r.Context(), listID, req.Name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to update list")
		return
	}
	h.hub.Broadcast(list.BoardID, models.WSEvent{Type: "list.updated", BoardID: list.BoardID, Payload: updated})
	JSON(w, http.StatusOK, updated)
}

func (h *BoardHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	listID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid list id")
		return
	}
	list, err := h.store.GetList(r.Context(), listID)
	if err != nil || list == nil {
		Error(w, http.StatusNotFound, "list not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), list.BoardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	if err := h.store.DeleteList(r.Context(), listID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete list")
		return
	}
	h.hub.Broadcast(list.BoardID, models.WSEvent{Type: "list.deleted", BoardID: list.BoardID, Payload: map[string]string{"id": listID.String()}})
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) ReorderLists(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BoardID uuid.UUID   `json:"board_id"`
		ListIDs []uuid.UUID `json:"list_ids"`
	}
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), req.BoardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	if err := h.store.ReorderLists(r.Context(), req.BoardID, req.ListIDs); err != nil {
		Error(w, http.StatusInternalServerError, "failed to reorder lists")
		return
	}
	h.hub.Broadcast(req.BoardID, models.WSEvent{Type: "lists.reordered", BoardID: req.BoardID, Payload: req.ListIDs})
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	listID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid list id")
		return
	}
	list, err := h.store.GetList(r.Context(), listID)
	if err != nil || list == nil {
		Error(w, http.StatusNotFound, "list not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), list.BoardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	userID, _, ok := h.requireMember(r, wsID)
	if !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	if err := Decode(r, &req); err != nil || strings.TrimSpace(req.Title) == "" {
		Error(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(req.Title) > 500 {
		Error(w, http.StatusBadRequest, "title too long")
		return
	}
	task, err := h.store.CreateTask(r.Context(), listID, req.Title, userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	h.hub.Broadcast(list.BoardID, models.WSEvent{Type: "task.created", BoardID: list.BoardID, Payload: task})
	JSON(w, http.StatusCreated, task)
}

func (h *BoardHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	boardID, err := h.store.GetTaskBoardID(r.Context(), taskID)
	if err != nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	task, err := h.store.GetTask(r.Context(), taskID)
	if err != nil || task == nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	JSON(w, http.StatusOK, task)
}

func (h *BoardHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	boardID, err := h.store.GetTaskBoardID(r.Context(), taskID)
	if err != nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		DueDate     *string `json:"due_date"`
	}
	if err := Decode(r, &req); err != nil || req.Title == "" {
		Error(w, http.StatusBadRequest, "title is required")
		return
	}
	task, err := h.store.UpdateTask(r.Context(), taskID, req.Title, req.Description, parseDueDate(req.DueDate))
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{Type: "task.updated", BoardID: boardID, Payload: task})
	JSON(w, http.StatusOK, task)
}

func (h *BoardHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	boardID, err := h.store.GetTaskBoardID(r.Context(), taskID)
	if err != nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	if err := h.store.DeleteTask(r.Context(), taskID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{Type: "task.deleted", BoardID: boardID, Payload: map[string]string{"id": taskID.String()}})
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) MoveTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	boardID, err := h.store.GetTaskBoardID(r.Context(), taskID)
	if err != nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	if _, _, ok := h.requireMember(r, wsID); !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		ListID   uuid.UUID `json:"list_id"`
		Position int       `json:"position"`
		ClientID string    `json:"client_id"`
	}
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	task, err := h.store.MoveTask(r.Context(), taskID, req.ListID, req.Position)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to move task")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{
		Type: "task.moved", BoardID: boardID, ClientID: req.ClientID, Payload: task,
	})
	JSON(w, http.StatusOK, task)
}

func (h *BoardHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	boardID, err := h.store.GetTaskBoardID(r.Context(), taskID)
	if err != nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	wsID, err := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	if err != nil {
		Error(w, http.StatusNotFound, "board not found")
		return
	}
	userID, _, ok := h.requireMember(r, wsID)
	if !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := Decode(r, &req); err != nil || strings.TrimSpace(req.Body) == "" {
		Error(w, http.StatusBadRequest, "body is required")
		return
	}
	body := htmlTagRe.ReplaceAllString(req.Body, "")
	comment, err := h.store.CreateComment(r.Context(), taskID, userID, body)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create comment")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{Type: "comment.created", BoardID: boardID, Payload: comment})
	h.notifyComment(r.Context(), taskID, boardID, userID, comment)
	JSON(w, http.StatusCreated, comment)
}

func (h *BoardHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid comment id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	var req struct {
		Body string `json:"body"`
	}
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	body := htmlTagRe.ReplaceAllString(req.Body, "")
	comment, err := h.store.UpdateComment(r.Context(), commentID, userID, body)
	if err != nil {
		Error(w, http.StatusNotFound, "comment not found")
		return
	}
	boardID, _ := h.store.GetTaskBoardID(r.Context(), comment.TaskID)
	h.hub.Broadcast(boardID, models.WSEvent{Type: "comment.updated", BoardID: boardID, Payload: comment})
	JSON(w, http.StatusOK, comment)
}

func (h *BoardHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid comment id")
		return
	}
	userID := middleware.UserIDFromContext(r.Context())
	if err := h.store.DeleteComment(r.Context(), commentID, userID); err != nil {
		Error(w, http.StatusNotFound, "comment not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) AddAssignee(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	boardID, err := h.store.GetTaskBoardID(r.Context(), taskID)
	if err != nil {
		Error(w, http.StatusNotFound, "task not found")
		return
	}
	wsID, _ := h.store.GetBoardWorkspaceID(r.Context(), boardID)
	actorID, _, ok := h.requireMember(r, wsID)
	if !ok {
		Error(w, http.StatusForbidden, "not a member")
		return
	}
	if err := h.store.AddTaskAssignee(r.Context(), taskID, userID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to add assignee")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{Type: "task.assignee.added", BoardID: boardID, Payload: map[string]string{"task_id": taskID.String(), "user_id": userID.String()}})
	h.notifyAssignment(r.Context(), taskID, boardID, actorID, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) RemoveAssignee(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user id")
		return
	}
	boardID, _ := h.store.GetTaskBoardID(r.Context(), taskID)
	if err := h.store.RemoveTaskAssignee(r.Context(), taskID, userID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to remove assignee")
		return
	}
	h.hub.Broadcast(boardID, models.WSEvent{Type: "task.assignee.removed", BoardID: boardID, Payload: map[string]string{"task_id": taskID.String(), "user_id": userID.String()}})
	w.WriteHeader(http.StatusNoContent)
}

func (h *BoardHandler) notifyAssignment(ctx context.Context, taskID, boardID, actorID, assigneeID uuid.UUID) {
	if h.notifier == nil || actorID == assigneeID {
		return
	}
	task, err := h.store.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return
	}
	actorName := "Someone"
	if actor, _ := h.store.GetUserByID(ctx, actorID); actor != nil {
		actorName = actor.Name
	}
	h.notifier.Push(ctx, assigneeID, "task_assigned", "Assigned to a task",
		fmt.Sprintf("%s assigned you to \"%s\"", actorName, task.Title),
		h.boardLink(ctx, boardID))
}

func (h *BoardHandler) notifyComment(ctx context.Context, taskID, boardID, actorID uuid.UUID, comment *models.Comment) {
	if h.notifier == nil {
		return
	}
	task, err := h.store.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return
	}
	actorName := "Someone"
	if comment.User != nil {
		actorName = comment.User.Name
	}

	recipients := make(map[uuid.UUID]struct{})
	if task.CreatedBy != uuid.Nil {
		recipients[task.CreatedBy] = struct{}{}
	}
	for _, a := range task.Assignees {
		recipients[a.ID] = struct{}{}
	}
	delete(recipients, actorID)

	link := h.boardLink(ctx, boardID)
	body := fmt.Sprintf("%s commented on \"%s\"", actorName, task.Title)
	for uid := range recipients {
		h.notifier.Push(ctx, uid, "comment_added", "New comment", body, link)
	}
}

func RegisterBoardRoutes(r chi.Router, h *BoardHandler) {
	r.Get("/workspaces/{id}/boards", h.ListBoards)
	r.Post("/workspaces/{id}/boards", h.CreateBoard)
	r.Get("/boards/{id}", h.GetBoard)
	r.Patch("/boards/{id}", h.UpdateBoard)
	r.Delete("/boards/{id}", h.DeleteBoard)
	r.Post("/boards/{id}/lists", h.CreateList)
	r.Patch("/lists/{id}", h.UpdateList)
	r.Delete("/lists/{id}", h.DeleteList)
	r.Patch("/lists/reorder", h.ReorderLists)
	r.Post("/lists/{id}/tasks", h.CreateTask)
	r.Get("/tasks/{id}", h.GetTask)
	r.Patch("/tasks/{id}", h.UpdateTask)
	r.Delete("/tasks/{id}", h.DeleteTask)
	r.Patch("/tasks/{id}/move", h.MoveTask)
	r.Post("/tasks/{id}/comments", h.CreateComment)
	r.Patch("/comments/{id}", h.UpdateComment)
	r.Delete("/comments/{id}", h.DeleteComment)
	r.Post("/tasks/{id}/assignees/{userId}", h.AddAssignee)
	r.Delete("/tasks/{id}/assignees/{userId}", h.RemoveAssignee)
}
