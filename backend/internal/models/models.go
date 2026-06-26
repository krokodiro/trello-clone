package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    *string    `json:"-"`
	Name            string     `json:"name"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	IsAdmin         bool       `json:"is_admin"`
	CreatedAt       time.Time  `json:"created_at"`
}

type AuthTokenType string

const (
	AuthTokenEmailVerification AuthTokenType = "email_verification"
	AuthTokenPasswordReset     AuthTokenType = "password_reset"
)

type AuthToken struct {
	ID        uuid.UUID     `json:"id"`
	UserID    uuid.UUID     `json:"user_id"`
	TokenHash string        `json:"-"`
	Type      AuthTokenType `json:"type"`
	ExpiresAt time.Time     `json:"expires_at"`
	UsedAt    *time.Time    `json:"used_at,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

type WorkspaceRole string

const (
	RoleOwner  WorkspaceRole = "owner"
	RoleAdmin  WorkspaceRole = "admin"
	RoleMember WorkspaceRole = "member"
)

type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkspaceMember struct {
	WorkspaceID uuid.UUID     `json:"workspace_id"`
	UserID      uuid.UUID     `json:"user_id"`
	Role        WorkspaceRole `json:"role"`
	JoinedAt    time.Time     `json:"joined_at"`
	User        *User         `json:"user,omitempty"`
}

type Invitation struct {
	ID          uuid.UUID     `json:"id"`
	WorkspaceID uuid.UUID     `json:"workspace_id"`
	Email       string        `json:"email"`
	Role        WorkspaceRole `json:"role"`
	Token       string        `json:"token,omitempty"`
	InvitedBy   uuid.UUID     `json:"invited_by"`
	ExpiresAt   time.Time     `json:"expires_at"`
	AcceptedAt  *time.Time    `json:"accepted_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

type Board struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type List struct {
	ID        uuid.UUID `json:"id"`
	BoardID   uuid.UUID `json:"board_id"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Tasks     []Task    `json:"tasks,omitempty"`
}

type Task struct {
	ID          uuid.UUID  `json:"id"`
	ListID      uuid.UUID  `json:"list_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Position    int        `json:"position"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Assignees   []User     `json:"assignees,omitempty"`
	Comments    []Comment  `json:"comments,omitempty"`
}

type Comment struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	UserID    uuid.UUID `json:"user_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `json:"user,omitempty"`
}

type BoardDetail struct {
	Board Board `json:"board"`
	Lists []List `json:"lists"`
}

type Notification struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type WSEvent struct {
	Type     string      `json:"type"`
	BoardID  uuid.UUID   `json:"board_id"`
	ClientID string      `json:"client_id,omitempty"`
	Payload  interface{} `json:"payload"`
}
