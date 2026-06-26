package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/trello-clone/backend/internal/auth"
	"github.com/trello-clone/backend/internal/config"
	"github.com/trello-clone/backend/internal/email"
	"github.com/trello-clone/backend/internal/middleware"
	"github.com/trello-clone/backend/internal/models"
	"github.com/trello-clone/backend/internal/store"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

const (
	emailVerificationDuration = 24 * time.Hour
	passwordResetDuration     = time.Hour
)

type AuthHandler struct {
	store  *store.Store
	tokens *auth.TokenManager
	cfg    *config.Config
	mailer *email.Mailer
}

func NewAuthHandler(s *store.Store, tm *auth.TokenManager, cfg *config.Config, mailer *email.Mailer) *AuthHandler {
	return &AuthHandler{store: s, tokens: tm, cfg: cfg, mailer: mailer}
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type emailReq struct {
	Email string `json:"email"`
}

type verifyEmailReq struct {
	Token string `json:"token"`
}

type resetPasswordReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		Error(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}
	if len(req.Password) < 8 {
		Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	existing, _ := h.store.GetUserByEmail(r.Context(), req.Email)
	if existing != nil {
		Error(w, http.StatusConflict, "email already registered")
		return
	}
	hash, err := h.tokens.HashPassword(req.Password)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	user, err := h.store.CreateUser(r.Context(), req.Email, req.Name, hash)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	link, err := h.sendVerificationEmail(r.Context(), user)
	if err != nil {
		log.Printf("verification email error: %v", err)
	}
	pair, err := h.issueTokens(r.Context(), user)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	resp := map[string]interface{}{
		"user":              sanitizeUser(user),
		"tokens":            pair,
		"verification_sent": h.mailer.Enabled(),
		"email_verified":    false,
	}
	if link != "" && !h.mailer.Enabled() {
		resp["verification_url"] = link
	}
	JSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || user == nil || user.PasswordHash == nil {
		Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := h.tokens.CheckPassword(*user.PasswordHash, req.Password); err != nil {
		Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if user.EmailVerifiedAt == nil {
		Error(w, http.StatusForbidden, "email not verified")
		return
	}
	pair, err := h.issueTokens(r.Context(), user)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	JSON(w, http.StatusOK, map[string]interface{}{"user": sanitizeUser(user), "tokens": pair})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	hash := auth.HashToken(req.RefreshToken)
	userID, expires, err := h.store.GetRefreshToken(r.Context(), hash)
	if err != nil || time.Now().After(expires) {
		Error(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	_ = h.store.DeleteRefreshToken(r.Context(), hash)
	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		Error(w, http.StatusUnauthorized, "user not found")
		return
	}
	pair, err := h.issueTokens(r.Context(), user)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	JSON(w, http.StatusOK, pair)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailReq
	if err := Decode(r, &req); err != nil || req.Token == "" {
		Error(w, http.StatusBadRequest, "token is required")
		return
	}
	if err := h.consumeAuthToken(r.Context(), req.Token, models.AuthTokenEmailVerification); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]string{"message": "email verified"})
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req emailReq
	if err := Decode(r, &req); err != nil || req.Email == "" {
		Error(w, http.StatusBadRequest, "email is required")
		return
	}
	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || user == nil || user.PasswordHash == nil {
		JSON(w, http.StatusOK, map[string]string{"message": "if the account exists, a verification email has been sent"})
		return
	}
	if user.EmailVerifiedAt != nil {
		JSON(w, http.StatusOK, map[string]string{"message": "email already verified"})
		return
	}
	if link, err := h.sendVerificationEmail(r.Context(), user); err != nil {
		log.Printf("verification email error: %v", err)
	} else if link != "" && !h.mailer.Enabled() {
		JSON(w, http.StatusOK, map[string]string{
			"message":          "if the account exists, a verification email has been sent",
			"verification_url": link,
		})
		return
	}
	JSON(w, http.StatusOK, map[string]string{"message": "if the account exists, a verification email has been sent"})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req emailReq
	if err := Decode(r, &req); err != nil || req.Email == "" {
		Error(w, http.StatusBadRequest, "email is required")
		return
	}
	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err == nil && user != nil && user.PasswordHash != nil {
		if err := h.sendPasswordResetEmail(r.Context(), user); err != nil {
			log.Printf("password reset email error: %v", err)
		}
	}
	JSON(w, http.StatusOK, map[string]string{"message": "if the account exists, a reset email has been sent"})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordReq
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" || req.Password == "" {
		Error(w, http.StatusBadRequest, "token and password are required")
		return
	}
	if len(req.Password) < 8 {
		Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	tokenHash := auth.HashToken(req.Token)
	authToken, err := h.store.GetAuthTokenByHash(r.Context(), tokenHash)
	if err != nil || authToken == nil {
		Error(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	if authToken.Type != models.AuthTokenPasswordReset {
		Error(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	if authToken.UsedAt != nil || time.Now().After(authToken.ExpiresAt) {
		Error(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	hash, err := h.tokens.HashPassword(req.Password)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if err := h.store.UpdatePassword(r.Context(), authToken.UserID, hash); err != nil {
		Error(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	if err := h.store.MarkAuthTokenUsed(r.Context(), tokenHash); err != nil {
		Error(w, http.StatusInternalServerError, "failed to complete reset")
		return
	}
	JSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func (h *AuthHandler) sendVerificationEmail(ctx context.Context, user *models.User) (string, error) {
	token, hash, expires, err := h.generateAuthToken(emailVerificationDuration)
	if err != nil {
		return "", err
	}
	if err := h.store.CreateAuthToken(ctx, user.ID, models.AuthTokenEmailVerification, hash, expires); err != nil {
		return "", err
	}
	link := fmt.Sprintf("%s/verify-email/%s", h.cfg.WebURL, token)
	body := fmt.Sprintf("Hi %s,\n\nVerify your email by opening this link:\n\n%s\n\nThis link expires in 24 hours.\n", user.Name, link)
	if err := h.mailer.Send(user.Email, "Verify your email", body); err != nil {
		return link, err
	}
	if !h.mailer.Enabled() {
		return link, nil
	}
	return "", nil
}

func (h *AuthHandler) sendPasswordResetEmail(ctx context.Context, user *models.User) error {
	token, hash, expires, err := h.generateAuthToken(passwordResetDuration)
	if err != nil {
		return err
	}
	if err := h.store.CreateAuthToken(ctx, user.ID, models.AuthTokenPasswordReset, hash, expires); err != nil {
		return err
	}
	link := fmt.Sprintf("%s/reset-password/%s", h.cfg.WebURL, token)
	body := fmt.Sprintf("Hi %s,\n\nReset your password by opening this link:\n\n%s\n\nThis link expires in 1 hour. If you did not request this, you can ignore this email.\n", user.Name, link)
	return h.mailer.Send(user.Email, "Reset your password", body)
}

func (h *AuthHandler) generateAuthToken(duration time.Duration) (string, string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", time.Time{}, err
	}
	token := hex.EncodeToString(b)
	hash := auth.HashToken(token)
	return token, hash, time.Now().Add(duration), nil
}

func (h *AuthHandler) consumeAuthToken(ctx context.Context, token string, tokenType models.AuthTokenType) error {
	tokenHash := auth.HashToken(token)
	authToken, err := h.store.GetAuthTokenByHash(ctx, tokenHash)
	if err != nil || authToken == nil {
		return fmt.Errorf("invalid or expired token")
	}
	if authToken.Type != tokenType {
		return fmt.Errorf("invalid or expired token")
	}
	if authToken.UsedAt != nil || time.Now().After(authToken.ExpiresAt) {
		return fmt.Errorf("invalid or expired token")
	}
	if tokenType == models.AuthTokenEmailVerification {
		if err := h.store.MarkEmailVerified(ctx, authToken.UserID); err != nil {
			return fmt.Errorf("failed to verify email")
		}
	}
	if err := h.store.MarkAuthTokenUsed(ctx, tokenHash); err != nil {
		return fmt.Errorf("failed to complete request")
	}
	return nil
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		Error(w, http.StatusNotFound, "user not found")
		return
	}
	JSON(w, http.StatusOK, sanitizeUser(user))
}

func (h *AuthHandler) issueTokens(ctx context.Context, user *models.User) (*models.TokenPair, error) {
	access, err := h.tokens.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}
	refresh, hash, expires, err := h.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := h.store.SaveRefreshToken(ctx, user.ID, hash, expires); err != nil {
		return nil, err
	}
	return &models.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func sanitizeUser(u *models.User) *models.User {
	u.PasswordHash = nil
	return u
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GoogleClientID == "" {
		Error(w, http.StatusNotImplemented, "Google OAuth not configured")
		return
	}
	conf := &oauth2.Config{
		ClientID:     h.cfg.GoogleClientID,
		ClientSecret: h.cfg.GoogleClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/auth", TokenURL: "https://oauth2.googleapis.com/token"},
	}
	state := uuid.New().String()
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", MaxAge: 300, HttpOnly: true})
	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, _ := r.Cookie("oauth_state")
	if stateCookie == nil || r.URL.Query().Get("state") != stateCookie.Value {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_state", http.StatusTemporaryRedirect)
		return
	}
	conf := &oauth2.Config{
		ClientID:     h.cfg.GoogleClientID,
		ClientSecret: h.cfg.GoogleClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/auth", TokenURL: "https://oauth2.googleapis.com/token"},
	}
	tok, err := conf.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_exchange", http.StatusTemporaryRedirect)
		return
	}
	h.oauthRedirect(w, r, "google", tok.AccessToken, conf)
}

func (h *AuthHandler) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GitHubClientID == "" {
		Error(w, http.StatusNotImplemented, "GitHub OAuth not configured")
		return
	}
	conf := h.githubConfig()
	state := uuid.New().String()
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state, Path: "/", MaxAge: 300, HttpOnly: true})
	http.Redirect(w, r, conf.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, _ := r.Cookie("oauth_state")
	if stateCookie == nil || r.URL.Query().Get("state") != stateCookie.Value {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_state", http.StatusTemporaryRedirect)
		return
	}
	conf := h.githubConfig()
	tok, err := conf.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_exchange", http.StatusTemporaryRedirect)
		return
	}
	client := conf.Client(r.Context(), tok)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_user", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_parse", http.StatusTemporaryRedirect)
		return
	}
	email := ghUser.Email
	if email == "" {
		email = fmt.Sprintf("%s@users.noreply.github.com", ghUser.Login)
	}
	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}
	h.finishOAuth(w, r, "github", fmt.Sprintf("%d", ghUser.ID), email, name, ghUser.AvatarURL)
}

func (h *AuthHandler) githubConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.cfg.GitHubClientID,
		ClientSecret: h.cfg.GitHubClientSecret,
		RedirectURL:  h.cfg.APIURL + "/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
}

func (h *AuthHandler) oauthRedirect(w http.ResponseWriter, r *http.Request, provider, accessToken string, conf *oauth2.Config) {
	client := conf.Client(r.Context(), &oauth2.Token{AccessToken: accessToken})
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_user", http.StatusTemporaryRedirect)
		return
	}
	defer resp.Body.Close()
	var info struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_parse", http.StatusTemporaryRedirect)
		return
	}
	h.finishOAuth(w, r, provider, info.ID, info.Email, info.Name, info.Picture)
}

func (h *AuthHandler) finishOAuth(w http.ResponseWriter, r *http.Request, provider, providerID, email, name, avatar string) {
	user, err := h.store.FindOrCreateOAuthUser(r.Context(), provider, providerID, email, name, avatar)
	if err != nil {
		log.Printf("oauth user error: %v", err)
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_create", http.StatusTemporaryRedirect)
		return
	}
	if user.EmailVerifiedAt == nil {
		_ = h.store.MarkEmailVerified(r.Context(), user.ID)
	}
	pair, err := h.issueTokens(r.Context(), user)
	if err != nil {
		http.Redirect(w, r, h.cfg.WebURL+"/login?error=oauth_token", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s", h.cfg.WebURL, pair.AccessToken, pair.RefreshToken), http.StatusTemporaryRedirect)
}

func RegisterAuthRoutes(r chi.Router, h *AuthHandler, authMw func(http.Handler) http.Handler) {
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.Refresh)
	r.Post("/auth/verify-email", h.VerifyEmail)
	r.Post("/auth/resend-verification", h.ResendVerification)
	r.Post("/auth/forgot-password", h.ForgotPassword)
	r.Post("/auth/reset-password", h.ResetPassword)
	r.Get("/auth/google", h.GoogleLogin)
	r.Get("/auth/google/callback", h.GoogleCallback)
	r.Get("/auth/github", h.GitHubLogin)
	r.Get("/auth/github/callback", h.GitHubCallback)
	r.With(authMw).Get("/auth/me", h.Me)
}
