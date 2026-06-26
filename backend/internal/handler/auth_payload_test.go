package handler

import "testing"

func TestAuthLinkPayload(t *testing.T) {
	t.Run("includes url when not sent", func(t *testing.T) {
		got := authLinkPayload("https://x/reset", false, "reset_url", "ok")
		if got["reset_url"] != "https://x/reset" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("omits url when sent", func(t *testing.T) {
		got := authLinkPayload("https://x/reset", true, "reset_url", "ok")
		if _, ok := got["reset_url"]; ok {
			t.Fatalf("expected no reset_url: %v", got)
		}
	})
}

func TestVerificationPayload(t *testing.T) {
	got := verificationPayload("https://x/verify", false, "sent")
	if got["verification_url"] != "https://x/verify" {
		t.Fatalf("got %v", got)
	}
}
