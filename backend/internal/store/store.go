package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trello-clone/backend/internal/models"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func Slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "workspace"
	}
	return s
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// --- Users ---

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, name, avatar_url, email_verified_at, is_admin, created_at`,
		email, name, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) EnsureAdminUser(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	existing, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		_, err := s.pool.Exec(ctx,
			`UPDATE users SET is_admin = TRUE, email_verified_at = COALESCE(email_verified_at, NOW()), updated_at = NOW() WHERE id = $1`,
			existing.ID,
		)
		if err != nil {
			return nil, err
		}
		return s.GetUserByID(ctx, existing.ID)
	}
	var u models.User
	err = s.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash, email_verified_at, is_admin)
		 VALUES ($1, $2, $3, NOW(), TRUE)
		 RETURNING id, email, password_hash, name, avatar_url, email_verified_at, is_admin, created_at`,
		email, name, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, avatar_url, email_verified_at, is_admin, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, avatar_url, email_verified_at, is_admin, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

func (s *Store) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, passwordHash,
	)
	return err
}

func (s *Store) CreateAuthToken(ctx context.Context, userID uuid.UUID, tokenType models.AuthTokenType, tokenHash string, expires time.Time) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM auth_tokens WHERE user_id = $1 AND type = $2 AND used_at IS NULL`,
		userID, tokenType,
	)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO auth_tokens (user_id, token_hash, type, expires_at) VALUES ($1, $2, $3, $4)`,
		userID, tokenHash, tokenType, expires,
	)
	return err
}

func (s *Store) GetAuthTokenByHash(ctx context.Context, tokenHash string) (*models.AuthToken, error) {
	var t models.AuthToken
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, type, expires_at, used_at, created_at
		 FROM auth_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Type, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (s *Store) MarkAuthTokenUsed(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE auth_tokens SET used_at = NOW() WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}

func (s *Store) SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expires time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expires,
	)
	return err
}

func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (uuid.UUID, time.Time, error) {
	var userID uuid.UUID
	var expires time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&userID, &expires)
	return userID, expires, err
}

func (s *Store) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) FindOrCreateOAuthUser(ctx context.Context, provider, providerUserID, email, name, avatarURL string) (*models.User, error) {
	var userID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&userID)
	if err == nil {
		return s.GetUserByID(ctx, userID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	existing, _ := s.GetUserByEmail(ctx, email)
	if existing != nil {
		userID = existing.ID
	} else {
		var avatar *string
		if avatarURL != "" {
			avatar = &avatarURL
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO users (email, name, avatar_url, email_verified_at) VALUES ($1, $2, $3, NOW())
			 RETURNING id`, email, name, avatar,
		).Scan(&userID)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_user_id) VALUES ($1, $2, $3)`,
		userID, provider, providerUserID,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

// --- Workspaces ---

func (s *Store) IsPlatformAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isAdmin bool
	err := s.pool.QueryRow(ctx, `SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&isAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return isAdmin, err
}

func (s *Store) ListAllWorkspaces(ctx context.Context) ([]models.Workspace, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, slug, owner_id, created_at, updated_at FROM workspaces ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Workspace
	for rows.Next() {
		var w models.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	admin, err := s.IsPlatformAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if admin {
		return s.ListAllWorkspaces(ctx)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT w.id, w.name, w.slug, w.owner_id, w.created_at, w.updated_at
		 FROM workspaces w
		 JOIN workspace_members wm ON wm.workspace_id = w.id
		 WHERE wm.user_id = $1 ORDER BY w.name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Workspace
	for rows.Next() {
		var w models.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) CreateWorkspace(ctx context.Context, name, slug string, ownerID uuid.UUID) (*models.Workspace, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var w models.Workspace
	err = tx.QueryRow(ctx,
		`INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $2, $3)
		 RETURNING id, name, slug, owner_id, created_at, updated_at`,
		name, slug, ownerID,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`,
		w.ID, ownerID,
	)
	if err != nil {
		return nil, err
	}
	return &w, tx.Commit(ctx)
}

func (s *Store) GetWorkspaceBySlug(ctx context.Context, slug string) (*models.Workspace, error) {
	var w models.Workspace
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, owner_id, created_at, updated_at FROM workspaces WHERE slug = $1`,
		slug,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &w, err
}

func (s *Store) GetWorkspaceByID(ctx context.Context, id uuid.UUID) (*models.Workspace, error) {
	var w models.Workspace
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, owner_id, created_at, updated_at FROM workspaces WHERE id = $1`,
		id,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &w, err
}

func (s *Store) UpdateWorkspace(ctx context.Context, id uuid.UUID, name string) (*models.Workspace, error) {
	var w models.Workspace
	err := s.pool.QueryRow(ctx,
		`UPDATE workspaces SET name = $2, updated_at = NOW() WHERE id = $1
		 RETURNING id, name, slug, owner_id, created_at, updated_at`,
		id, name,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.UpdatedAt)
	return &w, err
}

func (s *Store) DeleteWorkspace(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	return err
}

func (s *Store) GetMemberRole(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceRole, error) {
	admin, err := s.IsPlatformAdmin(ctx, userID)
	if err != nil {
		return "", err
	}
	if admin {
		return models.RoleOwner, nil
	}
	var role models.WorkspaceRole
	err = s.pool.QueryRow(ctx,
		`SELECT role FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	).Scan(&role)
	return role, err
}

func (s *Store) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceMember, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT wm.workspace_id, wm.user_id, wm.role, wm.joined_at,
		        u.id, u.email, u.name, u.avatar_url, u.email_verified_at, u.is_admin, u.created_at
		 FROM workspace_members wm
		 JOIN users u ON u.id = wm.user_id
		 WHERE wm.workspace_id = $1 ORDER BY wm.joined_at`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.WorkspaceMember
	for rows.Next() {
		var m models.WorkspaceMember
		var u models.User
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt,
			&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		m.User = &u
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role models.WorkspaceRole) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		workspaceID, userID, role,
	)
	return err
}

func (s *Store) UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role models.WorkspaceRole) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE workspace_members SET role = $3 WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID, role,
	)
	return err
}

func (s *Store) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	)
	return err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Store) GetPendingInvitation(ctx context.Context, workspaceID uuid.UUID, email string) (*models.Invitation, error) {
	email = normalizeEmail(email)
	var inv models.Invitation
	err := s.pool.QueryRow(ctx,
		`SELECT id, workspace_id, email, role, token, invited_by, expires_at, accepted_at, created_at
		 FROM invitations
		 WHERE workspace_id = $1 AND LOWER(email) = $2 AND accepted_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC LIMIT 1`,
		workspaceID, email,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &inv, err
}

func (s *Store) CreateInvitation(ctx context.Context, workspaceID uuid.UUID, email string, role models.WorkspaceRole, invitedBy uuid.UUID) (*models.Invitation, error) {
	email = normalizeEmail(email)
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	var inv models.Invitation
	err = s.pool.QueryRow(ctx,
		`INSERT INTO invitations (workspace_id, email, role, token, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '7 days')
		 RETURNING id, workspace_id, email, role, token, invited_by, expires_at, accepted_at, created_at`,
		workspaceID, email, role, token, invitedBy,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	return &inv, err
}

func (s *Store) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	var inv models.Invitation
	err := s.pool.QueryRow(ctx,
		`SELECT id, workspace_id, email, role, token, invited_by, expires_at, accepted_at, created_at
		 FROM invitations WHERE token = $1`,
		token,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &inv, err
}

func (s *Store) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.Workspace, error) {
	inv, err := s.GetInvitationByToken(ctx, token)
	if err != nil || inv == nil {
		return nil, errors.New("invitation not found")
	}
	if inv.AcceptedAt != nil {
		return nil, errors.New("invitation already accepted")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, errors.New("invitation expired")
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	if normalizeEmail(user.Email) != normalizeEmail(inv.Email) {
		return nil, errors.New("invitation was sent to a different email address")
	}

	if _, err := s.GetMemberRole(ctx, inv.WorkspaceID, userID); err == nil {
		_, _ = s.pool.Exec(ctx, `UPDATE invitations SET accepted_at = NOW() WHERE id = $1`, inv.ID)
		return s.GetWorkspaceByID(ctx, inv.WorkspaceID)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE invitations SET accepted_at = NOW() WHERE id = $1`, inv.ID)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, $3)`,
		inv.WorkspaceID, userID, inv.Role,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetWorkspaceByID(ctx, inv.WorkspaceID)
}

// --- Boards ---

func (s *Store) ListBoards(ctx context.Context, workspaceID uuid.UUID) ([]models.Board, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, workspace_id, name, description, position, created_at, updated_at
		 FROM boards WHERE workspace_id = $1 ORDER BY position, created_at`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Board
	for rows.Next() {
		var b models.Board
		if err := rows.Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.Description, &b.Position, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) CreateBoard(ctx context.Context, workspaceID uuid.UUID, name string, description *string) (*models.Board, error) {
	var maxPos int
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) FROM boards WHERE workspace_id = $1`, workspaceID).Scan(&maxPos)
	var b models.Board
	err := s.pool.QueryRow(ctx,
		`INSERT INTO boards (workspace_id, name, description, position) VALUES ($1, $2, $3, $4)
		 RETURNING id, workspace_id, name, description, position, created_at, updated_at`,
		workspaceID, name, description, maxPos+1,
	).Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.Description, &b.Position, &b.CreatedAt, &b.UpdatedAt)
	return &b, err
}

func (s *Store) GetBoard(ctx context.Context, id uuid.UUID) (*models.Board, error) {
	var b models.Board
	err := s.pool.QueryRow(ctx,
		`SELECT id, workspace_id, name, description, position, created_at, updated_at FROM boards WHERE id = $1`,
		id,
	).Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.Description, &b.Position, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &b, err
}

func (s *Store) UpdateBoard(ctx context.Context, id uuid.UUID, name string, description *string) (*models.Board, error) {
	var b models.Board
	err := s.pool.QueryRow(ctx,
		`UPDATE boards SET name = $2, description = $3, updated_at = NOW() WHERE id = $1
		 RETURNING id, workspace_id, name, description, position, created_at, updated_at`,
		id, name, description,
	).Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.Description, &b.Position, &b.CreatedAt, &b.UpdatedAt)
	return &b, err
}

func (s *Store) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM boards WHERE id = $1`, id)
	return err
}

func (s *Store) GetBoardWorkspaceID(ctx context.Context, boardID uuid.UUID) (uuid.UUID, error) {
	var wsID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT workspace_id FROM boards WHERE id = $1`, boardID).Scan(&wsID)
	return wsID, err
}

// --- Lists ---

func (s *Store) CreateList(ctx context.Context, boardID uuid.UUID, name string) (*models.List, error) {
	var maxPos int
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) FROM lists WHERE board_id = $1`, boardID).Scan(&maxPos)
	var l models.List
	err := s.pool.QueryRow(ctx,
		`INSERT INTO lists (board_id, name, position) VALUES ($1, $2, $3)
		 RETURNING id, board_id, name, position, created_at, updated_at`,
		boardID, name, maxPos+1,
	).Scan(&l.ID, &l.BoardID, &l.Name, &l.Position, &l.CreatedAt, &l.UpdatedAt)
	return &l, err
}

func (s *Store) GetList(ctx context.Context, id uuid.UUID) (*models.List, error) {
	var l models.List
	err := s.pool.QueryRow(ctx,
		`SELECT id, board_id, name, position, created_at, updated_at FROM lists WHERE id = $1`,
		id,
	).Scan(&l.ID, &l.BoardID, &l.Name, &l.Position, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &l, err
}

func (s *Store) UpdateList(ctx context.Context, id uuid.UUID, name string) (*models.List, error) {
	var l models.List
	err := s.pool.QueryRow(ctx,
		`UPDATE lists SET name = $2, updated_at = NOW() WHERE id = $1
		 RETURNING id, board_id, name, position, created_at, updated_at`,
		id, name,
	).Scan(&l.ID, &l.BoardID, &l.Name, &l.Position, &l.CreatedAt, &l.UpdatedAt)
	return &l, err
}

func (s *Store) DeleteList(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM lists WHERE id = $1`, id)
	return err
}

func (s *Store) ReorderLists(ctx context.Context, boardID uuid.UUID, listIDs []uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range listIDs {
		_, err := tx.Exec(ctx, `UPDATE lists SET position = $2, updated_at = NOW() WHERE id = $1 AND board_id = $3`, id, i, boardID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// --- Tasks ---

func (s *Store) CreateTask(ctx context.Context, listID uuid.UUID, title string, createdBy uuid.UUID) (*models.Task, error) {
	var maxPos int
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) FROM tasks WHERE list_id = $1`, listID).Scan(&maxPos)
	var t models.Task
	err := s.pool.QueryRow(ctx,
		`INSERT INTO tasks (list_id, title, position, created_by) VALUES ($1, $2, $3, $4)
		 RETURNING id, list_id, title, description, position, due_date, created_by, created_at, updated_at`,
		listID, title, maxPos+1, createdBy,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Position, &t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	return &t, err
}

func (s *Store) GetTask(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx,
		`SELECT id, list_id, title, description, position, due_date, created_by, created_at, updated_at
		 FROM tasks WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Position, &t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Assignees, _ = s.ListTaskAssignees(ctx, id)
	t.Comments, _ = s.ListComments(ctx, id)
	if t.Assignees == nil {
		t.Assignees = []models.User{}
	}
	if t.Comments == nil {
		t.Comments = []models.Comment{}
	}
	return &t, nil
}

func (s *Store) UpdateTask(ctx context.Context, id uuid.UUID, title string, description *string, dueDate *time.Time) (*models.Task, error) {
	var t models.Task
	err := s.pool.QueryRow(ctx,
		`UPDATE tasks SET title = $2, description = $3, due_date = $4, updated_at = NOW() WHERE id = $1
		 RETURNING id, list_id, title, description, position, due_date, created_by, created_at, updated_at`,
		id, title, description, dueDate,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Position, &t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	return &t, err
}

func (s *Store) DeleteTask(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	return err
}

func (s *Store) MoveTask(ctx context.Context, taskID, listID uuid.UUID, position int) (*models.Task, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var oldListID uuid.UUID
	var oldPos int
	err = tx.QueryRow(ctx, `SELECT list_id, position FROM tasks WHERE id = $1`, taskID).Scan(&oldListID, &oldPos)
	if err != nil {
		return nil, err
	}

	if oldListID == listID {
		if position > oldPos {
			_, err = tx.Exec(ctx,
				`UPDATE tasks SET position = position - 1, updated_at = NOW()
				 WHERE list_id = $1 AND position > $2 AND position <= $3`,
				listID, oldPos, position,
			)
		} else if position < oldPos {
			_, err = tx.Exec(ctx,
				`UPDATE tasks SET position = position + 1, updated_at = NOW()
				 WHERE list_id = $1 AND position >= $2 AND position < $3`,
				listID, position, oldPos,
			)
		}
	} else {
		_, err = tx.Exec(ctx,
			`UPDATE tasks SET position = position - 1, updated_at = NOW()
			 WHERE list_id = $1 AND position > $2`, oldListID, oldPos,
		)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx,
			`UPDATE tasks SET position = position + 1, updated_at = NOW()
			 WHERE list_id = $1 AND position >= $2`, listID, position,
		)
	}
	if err != nil {
		return nil, err
	}

	var t models.Task
	err = tx.QueryRow(ctx,
		`UPDATE tasks SET list_id = $2, position = $3, updated_at = NOW() WHERE id = $1
		 RETURNING id, list_id, title, description, position, due_date, created_by, created_at, updated_at`,
		taskID, listID, position,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Position, &t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, tx.Commit(ctx)
}

func (s *Store) GetTaskBoardID(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error) {
	var boardID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT l.board_id FROM tasks t JOIN lists l ON l.id = t.list_id WHERE t.id = $1`,
		taskID,
	).Scan(&boardID)
	return boardID, err
}

func (s *Store) GetBoardDetail(ctx context.Context, boardID uuid.UUID) (*models.BoardDetail, error) {
	board, err := s.GetBoard(ctx, boardID)
	if err != nil || board == nil {
		return nil, err
	}

	listRows, err := s.pool.Query(ctx,
		`SELECT id, board_id, name, position, created_at, updated_at
		 FROM lists WHERE board_id = $1 ORDER BY position`,
		boardID,
	)
	if err != nil {
		return nil, err
	}
	defer listRows.Close()

	var lists []models.List
	for listRows.Next() {
		var l models.List
		if err := listRows.Scan(&l.ID, &l.BoardID, &l.Name, &l.Position, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		taskRows, err := s.pool.Query(ctx,
			`SELECT id, list_id, title, description, position, due_date, created_by, created_at, updated_at
			 FROM tasks WHERE list_id = $1 ORDER BY position`,
			l.ID,
		)
		if err != nil {
			return nil, err
		}
		for taskRows.Next() {
			var t models.Task
			if err := taskRows.Scan(&t.ID, &t.ListID, &t.Title, &t.Description, &t.Position, &t.DueDate, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
				taskRows.Close()
				return nil, err
			}
			t.Assignees, _ = s.ListTaskAssignees(ctx, t.ID)
			l.Tasks = append(l.Tasks, t)
		}
		taskRows.Close()
		lists = append(lists, l)
	}

	if lists == nil {
		lists = []models.List{}
	}

	return &models.BoardDetail{Board: *board, Lists: lists}, nil
}

// --- Comments ---

func (s *Store) ListComments(ctx context.Context, taskID uuid.UUID) ([]models.Comment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, c.task_id, c.user_id, c.body, c.created_at, c.updated_at,
		        u.id, u.email, u.name, u.avatar_url, u.email_verified_at, u.is_admin, u.created_at
		 FROM comments c JOIN users u ON u.id = c.user_id
		 WHERE c.task_id = $1 ORDER BY c.created_at`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Comment
	for rows.Next() {
		var c models.Comment
		var u models.User
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt,
			&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		c.User = &u
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateComment(ctx context.Context, taskID, userID uuid.UUID, body string) (*models.Comment, error) {
	var c models.Comment
	err := s.pool.QueryRow(ctx,
		`INSERT INTO comments (task_id, user_id, body) VALUES ($1, $2, $3)
		 RETURNING id, task_id, user_id, body, created_at, updated_at`,
		taskID, userID, body,
	).Scan(&c.ID, &c.TaskID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u, _ := s.GetUserByID(ctx, userID)
	c.User = u
	return &c, nil
}

func (s *Store) UpdateComment(ctx context.Context, id, userID uuid.UUID, body string) (*models.Comment, error) {
	var c models.Comment
	err := s.pool.QueryRow(ctx,
		`UPDATE comments SET body = $3, updated_at = NOW() WHERE id = $1 AND user_id = $2
		 RETURNING id, task_id, user_id, body, created_at, updated_at`,
		id, userID, body,
	).Scan(&c.ID, &c.TaskID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func (s *Store) DeleteComment(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM comments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}

// --- Assignees ---

func (s *Store) AddTaskAssignee(ctx context.Context, taskID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO task_assignees (task_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		taskID, userID,
	)
	return err
}

func (s *Store) RemoveTaskAssignee(ctx context.Context, taskID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM task_assignees WHERE task_id = $1 AND user_id = $2`, taskID, userID)
	return err
}

func (s *Store) ListTaskAssignees(ctx context.Context, taskID uuid.UUID) ([]models.User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.email, u.name, u.avatar_url, u.email_verified_at, u.is_admin, u.created_at
		 FROM users u JOIN task_assignees ta ON ta.user_id = u.id WHERE ta.task_id = $1`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.EmailVerifiedAt, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// --- Notifications ---

func (s *Store) CreateNotification(ctx context.Context, userID uuid.UUID, ntype, title, body, link string) (*models.Notification, error) {
	var n models.Notification
	err := s.pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, type, title, body, link) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, type, title, body, link, read, created_at`,
		userID, ntype, title, body, link,
	).Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.Read, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *Store) ListNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, type, title, body, link, read, created_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`,
		userID,
	).Scan(&count)
	return count, err
}

func (s *Store) MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`,
		userID,
	)
	return err
}

func (s *Store) GetBoardSlug(ctx context.Context, boardID uuid.UUID) (string, error) {
	var slug string
	err := s.pool.QueryRow(ctx,
		`SELECT w.slug FROM boards b JOIN workspaces w ON w.id = b.workspace_id WHERE b.id = $1`,
		boardID,
	).Scan(&slug)
	return slug, err
}

func CanManage(role models.WorkspaceRole) bool {
	return role == models.RoleOwner || role == models.RoleAdmin
}
