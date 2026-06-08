package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type Client struct {
	host      string
	port      string
	fromEmail string
}

func NewClient(host, port, fromEmail string) *Client {
	return &Client{
		host:      host,
		port:      port,
		fromEmail: fromEmail,
	}
}

func (c *Client) Send(ctx context.Context, to, subject, body string) error {
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", c.fromEmail, to, subject, body)

	addr := fmt.Sprintf("%s:%s", c.host, c.port)
	return smtp.SendMail(addr, nil, c.fromEmail, []string{to}, []byte(msg))
}
