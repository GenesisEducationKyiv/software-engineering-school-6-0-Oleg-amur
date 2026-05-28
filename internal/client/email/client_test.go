package email

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestClientSendTimesOutWhenSMTPServerDoesNotRespond(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	done := make(chan struct{})
	defer close(done)

	accepted := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		close(accepted)
		defer conn.Close()
		<-done
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}

	client := NewClient("127.0.0.1", port, "from@example.com", 20*time.Millisecond)

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
