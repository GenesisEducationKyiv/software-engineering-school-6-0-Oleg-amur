package email

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

type Client struct {
	host      string
	port      string
	fromEmail string
	timeout   time.Duration
}

func NewClient(host, port, fromEmail string, timeout time.Duration) *Client {
	return &Client{
		host:      host,
		port:      port,
		fromEmail: fromEmail,
		timeout:   timeout,
	}
}

func (c *Client) Send(ctx context.Context, to, subject, body string) error {
	if ctx == nil {
		ctx = context.Background()
	}

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

	return sendMail(conn, c.host, c.fromEmail, []string{to}, []byte(msg))
}

func sendMail(conn net.Conn, host, from string, to []string, msg []byte) error {
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

	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit smtp session: %w", err)
	}

	return nil
}
