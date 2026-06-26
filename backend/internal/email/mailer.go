package email

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/trello-clone/backend/internal/config"
)

type Mailer struct {
	enabled bool
	host    string
	port    string
	user    string
	pass    string
	from    string
}

func New(cfg *config.Config) *Mailer {
	enabled := cfg.SMTPHost != ""
	return &Mailer{
		enabled: enabled,
		host:    cfg.SMTPHost,
		port:    cfg.SMTPPort,
		user:    cfg.SMTPUser,
		pass:    cfg.SMTPPassword,
		from:    cfg.SMTPFrom,
	}
}

func (m *Mailer) Send(to, subject, body string) error {
	if !m.enabled {
		log.Printf("[email] SMTP not configured — would send to %s\nSubject: %s\n%s", to, subject, body)
		return nil
	}

	if err := m.send(to, subject, body); err != nil {
		log.Printf("[email] send failed to %s: %v", to, err)
		return err
	}
	log.Printf("[email] sent to %s: %s", to, subject)
	return nil
}

func (m *Mailer) send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", m.from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
