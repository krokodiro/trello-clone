package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/trello-clone/backend/internal/auth"
)

func TestHashPasswordAndCheck(t *testing.T) {
	tm := auth.NewTokenManager("test-secret")
	hash, err := tm.HashPassword("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if err := tm.CheckPassword(hash, "correct-horse"); err != nil {
		t.Fatalf("expected password to match: %v", err)
	}
	if err := tm.CheckPassword(hash, "wrong"); err == nil {
		t.Fatal("expected wrong password to fail")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	tm := auth.NewTokenManager("jwt-test-secret")
	userID := uuid.New()
	email := "user@example.com"

	token, err := tm.GenerateAccessToken(userID, email)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.ParseAccessToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != userID {
		t.Fatalf("user id: got %v want %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Fatalf("email: got %q want %q", claims.Email, email)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		t.Fatal("expected future expiry")
	}
}

func TestParseAccessTokenRejectsWrongSecret(t *testing.T) {
	tm := auth.NewTokenManager("secret-a")
	other := auth.NewTokenManager("secret-b")
	token, err := tm.GenerateAccessToken(uuid.New(), "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ParseAccessToken(token); err == nil {
		t.Fatal("expected invalid token with wrong secret")
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	a := auth.HashToken("abc")
	b := auth.HashToken("abc")
	if a != b {
		t.Fatalf("hash not deterministic: %q vs %q", a, b)
	}
	if auth.HashToken("xyz") == a {
		t.Fatal("different tokens should hash differently")
	}
}

func TestGenerateRefreshTokenUnique(t *testing.T) {
	tm := auth.NewTokenManager("s")
	t1, h1, _, err := tm.GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	t2, h2, _, err := tm.GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if t1 == t2 || h1 == h2 {
		t.Fatal("expected unique refresh tokens")
	}
}
