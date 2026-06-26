package email

import (
	"log"
	"strings"

	"github.com/trello-clone/backend/internal/config"
)

type sender interface {
	send(to, subject, body string) error
}

type Mailer struct {
	enabled bool
	mode    string
	impl    sender
}

func New(cfg *config.Config) *Mailer {
	from := cfg.EmailFrom
	if from == "" {
		from = "noreply@localhost"
	}

	if key := strings.TrimSpace(cfg.ResendAPIKey); key != "" {
		log.Printf("[email] using Resend HTTP API (from %q)", from)
		return &Mailer{
			enabled: true,
			mode:    "resend",
			impl:    newResendSender(key, from),
		}
	}

	if cfg.SMTPHost != "" {
		log.Printf("[email] using SMTP %s:%s (from %q)", cfg.SMTPHost, cfg.SMTPPort, from)
		return &Mailer{
			enabled: true,
			mode:    "smtp",
			impl: newSMTPSender(
				cfg.SMTPHost,
				cfg.SMTPPort,
				cfg.SMTPUser,
				cfg.SMTPPassword,
				from,
			),
		}
	}

	return &Mailer{enabled: false, mode: "none"}
}

func (m *Mailer) Enabled() bool {
	return m.enabled
}

func (m *Mailer) Send(to, subject, body string) error {
	if !m.enabled {
		log.Printf("[email] not configured — would send to %s: %s", to, subject)
		return nil
	}

	if err := m.impl.send(to, subject, body); err != nil {
		log.Printf("[email] send failed (%s) to %s: %v", m.mode, to, err)
		return err
	}
	log.Printf("[email] sent (%s) to %s: %s", m.mode, to, subject)
	return nil
}
