package notifier

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/config"
)

type EmailNotifier struct {
	cfg config.Notifier
}

func NewEmailNotifier(cfg config.Notifier) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (n *EmailNotifier) SendConfirmation(ctx context.Context, email, token string) error {
	subject := "Confirm your subscription"
	body := fmt.Sprintf("Please confirm your subscription by clicking here: %s/%s", n.cfg.ConfirmationUrl, token)
	return n.send(email, subject, body)
}

func (n *EmailNotifier) SendReleaseNotification(ctx context.Context, email, repo, tag string) error {
	subject := fmt.Sprintf("New release for %s", repo)
	body := fmt.Sprintf("A new release %s is available for %s!", tag, repo)
	return n.send(email, subject, body)
}

func (n *EmailNotifier) send(to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", n.cfg.FromEmail, to, subject, body)

	addr := fmt.Sprintf("%s:%s", n.cfg.SMTPHost, n.cfg.SMTPPort)

	return smtp.SendMail(addr, nil, n.cfg.FromEmail, []string{to}, []byte(msg))
}
