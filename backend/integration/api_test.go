//go:build integration

package integration_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/trello-clone/backend/internal/testutil"
)

func TestAuthRegisterVerifyLogin(t *testing.T) {
	api := testutil.SetupAPI(t)
	email := "user-" + uuid.New().String()[:8] + "@example.com"

	resp := api.Do("POST", "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": "password123",
		"name":     "Test User",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: %d", resp.StatusCode)
	}
	var reg struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
		VerificationURL string `json:"verification_url"`
	}
	api.Decode(resp, &reg)
	if reg.VerificationURL == "" {
		t.Fatal("expected verification_url when mailer is off")
	}
	token := strings.TrimPrefix(reg.VerificationURL, "http://localhost:3000/verify-email/")

	resp = api.Do("POST", "/api/v1/auth/verify-email", map[string]string{"token": token}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify: %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = api.Do("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "password123",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d", resp.StatusCode)
	}
	var login struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	api.Decode(resp, &login)

	resp = api.Do("GET", "/api/v1/auth/me", nil, login.Tokens.AccessToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestForgotPasswordWithoutEmail(t *testing.T) {
	api := testutil.SetupAPI(t)
	email := "reset-" + uuid.New().String()[:8] + "@example.com"

	resp := api.Do("POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": "password123", "name": "Reset User",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: %d", resp.StatusCode)
	}
	var reg struct {
		VerificationURL string `json:"verification_url"`
	}
	api.Decode(resp, &reg)
	token := strings.TrimPrefix(reg.VerificationURL, "http://localhost:3000/verify-email/")
	resp = api.Do("POST", "/api/v1/auth/verify-email", map[string]string{"token": token}, "")
	resp.Body.Close()

	resp = api.Do("POST", "/api/v1/auth/forgot-password", map[string]string{"email": email}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("forgot: %d", resp.StatusCode)
	}
	var forgot struct {
		ResetURL string `json:"reset_url"`
	}
	api.Decode(resp, &forgot)
	if forgot.ResetURL == "" {
		t.Fatal("expected reset_url")
	}
	resetToken := strings.TrimPrefix(forgot.ResetURL, "http://localhost:3000/reset-password/")

	resp = api.Do("POST", "/api/v1/auth/reset-password", map[string]string{
		"token":    resetToken,
		"password": "newpassword99",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reset: %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = api.Do("POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": "newpassword99",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login with new password: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestWorkspaceCRUD(t *testing.T) {
	api := testutil.SetupAPI(t)
	token := registerVerifiedUser(t, api)

	resp := api.Do("POST", "/api/v1/workspaces", map[string]string{"name": "Engineering"}, token)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create workspace: %d", resp.StatusCode)
	}
	var ws struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	api.Decode(resp, &ws)

	resp = api.Do("GET", "/api/v1/workspaces", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list workspaces: %d", resp.StatusCode)
	}
	var list []struct {
		ID string `json:"id"`
	}
	api.Decode(resp, &list)
	if len(list) != 1 || list[0].ID != ws.ID {
		t.Fatalf("unexpected workspaces: %+v", list)
	}
}

func registerVerifiedUser(t *testing.T, api *testutil.APIClient) string {
	t.Helper()
	email := "ws-" + uuid.New().String()[:8] + "@example.com"
	resp := api.Do("POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": "password123", "name": "WS User",
	}, "")
	var reg struct {
		VerificationURL string `json:"verification_url"`
		Tokens          struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	api.Decode(resp, &reg)
	token := strings.TrimPrefix(reg.VerificationURL, "http://localhost:3000/verify-email/")
	resp = api.Do("POST", "/api/v1/auth/verify-email", map[string]string{"token": token}, "")
	resp.Body.Close()
	resp = api.Do("POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": "password123",
	}, "")
	var login struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	api.Decode(resp, &login)
	return login.Tokens.AccessToken
}
