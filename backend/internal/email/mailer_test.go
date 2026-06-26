package email

import "testing"

func TestExtractEmail(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"noreply@example.com", "noreply@example.com"},
		{"Trello <noreply@example.com>", "noreply@example.com"},
		{"  user@test.com  ", "user@test.com"},
	}
	for _, tc := range tests {
		if got := extractEmail(tc.in); got != tc.want {
			t.Errorf("extractEmail(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMailerDisabledSendNoop(t *testing.T) {
	m := &Mailer{enabled: false, mode: "none"}
	if err := m.Send("a@b.com", "sub", "body"); err != nil {
		t.Fatal(err)
	}
	if m.Enabled() {
		t.Fatal("expected disabled")
	}
}
