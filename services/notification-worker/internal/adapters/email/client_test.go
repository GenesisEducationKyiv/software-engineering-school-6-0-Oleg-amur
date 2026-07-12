package email

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestIsTransientSendError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantTransient bool
	}{
		{
			name:          "SMTP 421 is transient",
			err:           &textproto.Error{Code: 421, Msg: "service unavailable"},
			wantTransient: true,
		},
		{
			name: "wrapped SMTP 550 is permanent",
			err: fmt.Errorf(
				"set smtp recipient: %w",
				&textproto.Error{Code: 550, Msg: "mailbox unavailable"},
			),
			wantTransient: false,
		},
		{
			name:          "unknown transport error is transient",
			err:           errors.New("connection reset"),
			wantTransient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTransientSendError(tt.err); got != tt.wantTransient {
				t.Errorf("isTransientSendError() = %t, want %t", got, tt.wantTransient)
			}
		})
	}
}

func TestClientSendTimesOutWhenSMTPServerDoesNotRespond(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	done := make(chan struct{})
	defer close(done)

	accepted := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		close(accepted)
		defer func() {
			_ = conn.Close()
		}()
		<-done
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}

	client := NewClient(
		testLogger(),
		"127.0.0.1",
		port,
		"from@example.com",
		20*time.Millisecond,
		testRetryPolicy(t, 1),
	)

	start := time.Now()
	err = client.Send(context.Background(), "to@example.com", "subject", "body")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > time.Second {
		t.Fatalf("expected send to return quickly, took %s", elapsed)
	}

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("expected smtp connection to be accepted")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testRetryPolicy(t *testing.T, maxAttempts int) RetryPolicy {
	t.Helper()

	policy, err := NewRetryPolicy(maxAttempts, time.Millisecond, time.Second)
	if err != nil {
		t.Fatalf("create retry policy: %v", err)
	}
	policy.waitForBackoff = func(context.Context, time.Duration) error { return nil }
	return policy
}

func TestSendMailTreatsQuitFailureAsSuccessAfterMessageWasAccepted(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	serverDone := make(chan error, 1)
	go func() {
		defer func() {
			_ = serverConn.Close()
		}()
		serverDone <- runSMTPServerUntilQuit(serverConn)
	}()

	err := sendMail(
		log,
		clientConn,
		"localhost",
		"from@example.com",
		[]string{"to@example.com"},
		[]byte("Subject: test\r\n\r\nbody\r\n"),
	)
	if err != nil {
		t.Fatalf("sendMail returned error after SMTP server accepted message: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake SMTP server failed: %v", err)
	}
	if !strings.Contains(logs.String(), "failed to close SMTP session after message was accepted") {
		t.Fatalf("expected SMTP session close failure to be logged, got %q", logs.String())
	}
}

func runSMTPServerUntilQuit(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writeResponse := func(response string) error {
		if _, err := writer.WriteString(response + "\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}
	readCommand := func(prefix string) error {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if !strings.HasPrefix(line, prefix) {
			return fmt.Errorf("got SMTP command %q, want prefix %q", line, prefix)
		}
		return nil
	}

	if err := writeResponse("220 localhost ESMTP ready"); err != nil {
		return err
	}
	for _, exchange := range []struct {
		command  string
		response string
	}{
		{command: "EHLO ", response: "250 localhost"},
		{command: "MAIL FROM:", response: "250 sender accepted"},
		{command: "RCPT TO:", response: "250 recipient accepted"},
		{command: "DATA", response: "354 send message"},
	} {
		if err := readCommand(exchange.command); err != nil {
			return err
		}
		if err := writeResponse(exchange.response); err != nil {
			return err
		}
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if line == ".\r\n" {
			break
		}
	}
	if err := writeResponse("250 message accepted"); err != nil {
		return err
	}

	// Read QUIT and close the connection without sending its response. The
	// message was already accepted, so this session-close failure is harmless.
	return readCommand("QUIT")
}
