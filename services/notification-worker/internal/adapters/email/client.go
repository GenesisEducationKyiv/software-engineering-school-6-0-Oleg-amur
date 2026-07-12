package email

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"net/textproto"
	"time"
)

type Client struct {
	log         *slog.Logger
	host        string
	port        string
	fromEmail   string
	timeout     time.Duration
	retryPolicy RetryPolicy
}

func NewClient(
	log *slog.Logger,
	host, port, fromEmail string,
	timeout time.Duration,
	retryPolicy RetryPolicy,
) *Client {
	return &Client{
		log:         log,
		host:        host,
		port:        port,
		fromEmail:   fromEmail,
		timeout:     timeout,
		retryPolicy: retryPolicy,
	}
}

func (c *Client) Send(ctx context.Context, to, subject, body string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return c.retryPolicy.execute(
		ctx,
		func() error { return c.sendOnce(ctx, to, subject, body) },
		isTransientSendError,
		func(attempt int, delay time.Duration, err error) {
			c.log.Warn(
				"temporary email delivery failure; retrying",
				"email", to,
				"attempt", attempt,
				"next_attempt", attempt+1,
				"retry_in", delay,
				"err", err,
			)
		},
	)
}

func (c *Client) sendOnce(ctx context.Context, to, subject, body string) error {
	sendCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", c.fromEmail, to, subject, body)

	addr := net.JoinHostPort(c.host, c.port)

	var dialer net.Dialer
	conn, err := dialer.DialContext(sendCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect to smtp server: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	if deadline, ok := sendCtx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("set smtp deadline: %w", err)
		}
	}

	return sendMail(c.log, conn, c.host, c.fromEmail, []string{to}, []byte(msg))
}

func isTransientSendError(err error) bool {
	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		return smtpErr.Code >= 400 && smtpErr.Code < 500
	}

	// Network, timeout and unknown transport errors are treated as transient.
	// It is safer to retry an unclassified delivery failure than to compensate
	// a subscription that could still receive its confirmation email.
	return true
}

func sendMail(log *slog.Logger, conn net.Conn, host, from string, to []string, msg []byte) error {
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set smtp sender: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("set smtp recipient: %w", err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start smtp data: %w", err)
	}

	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp message: %w", err)
	}

	// A successful DATA writer close means that the SMTP server accepted the
	// message. A failure while closing the session must not turn that accepted
	// delivery into a retry, which could send the same email more than once.
	if err := client.Quit(); err != nil {
		log.Warn("failed to close SMTP session after message was accepted", "err", err)
	}

	return nil
}
