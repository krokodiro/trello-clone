//go:build integration

package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/trello-clone/backend/internal/auth"
	"github.com/trello-clone/backend/internal/config"
	"github.com/trello-clone/backend/internal/database"
	"github.com/trello-clone/backend/internal/email"
	"github.com/trello-clone/backend/internal/handler"
	authmw "github.com/trello-clone/backend/internal/middleware"
	"github.com/trello-clone/backend/internal/store"
	"github.com/trello-clone/backend/internal/ws"
)

type APIClient struct {
	Server *httptest.Server
	Store  *store.Store
	Tokens *auth.TokenManager
	t      *testing.T
}

func (c *APIClient) Do(method, path string, body interface{}, bearer string) *http.Response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, c.Server.URL+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	return resp
}

func (c *APIClient) Decode(resp *http.Response, v interface{}) {
	c.t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		c.t.Fatal(err)
	}
}

func DatabaseURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://trello:trello@localhost:5432/trello?sslmode=disable"
}

func ModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func SetupAPI(t *testing.T) *APIClient {
	t.Helper()
	ctx := context.Background()
	dbURL := DatabaseURL()
	pool, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	root := ModuleRoot(t)
	if err := database.RunMigrationsFrom(dbURL, filepath.Join(root, "migrations")); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, oauth_accounts, workspaces, workspace_members, boards, lists, tasks, task_assignees, comments, invitations, refresh_tokens, auth_tokens, notifications RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	st := store.New(pool)
	tm := auth.NewTokenManager("integration-test-secret")
	cfg := &config.Config{
		WebURL: "http://localhost:3000",
		APIURL: "http://localhost:8080",
	}
	mailer := email.New(cfg)
	hub := ws.NewHub(nil, tm)
	authH := handler.NewAuthHandler(st, tm, cfg, mailer)
	notifH := handler.NewNotificationHandler(st, hub)
	wsH := handler.NewWorkspaceHandler(st, cfg, hub, mailer, notifH)
	boardH := handler.NewBoardHandler(st, hub, notifH)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	authMiddleware := authmw.Auth(tm)
	verifiedMiddleware := authmw.RequireVerified(st)

	r.Route("/api/v1", func(api chi.Router) {
		handler.RegisterAuthRoutes(api, authH, authMiddleware)
		api.Group(func(protected chi.Router) {
			protected.Use(authMiddleware)
			protected.Use(verifiedMiddleware)
			handler.RegisterWorkspaceRoutes(protected, wsH)
			handler.RegisterBoardRoutes(protected, boardH)
		})
		api.Group(func(authOnly chi.Router) {
			authOnly.Use(authMiddleware)
			handler.RegisterInvitationRoutes(authOnly, wsH)
			handler.RegisterNotificationRoutes(authOnly, notifH)
		})
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &APIClient{Server: srv, Store: st, Tokens: tm, t: t}
}
