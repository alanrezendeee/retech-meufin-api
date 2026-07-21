package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestInProcess_DispatchesToHandler(t *testing.T) {
	q := NewInProcess(2, 16, 0, time.Millisecond, nil)
	done := make(chan Message, 1)
	q.Register("t", func(_ context.Context, msg Message) error {
		done <- msg
		return nil
	})
	q.Start(context.Background())

	_ = q.Publish(context.Background(), Message{Type: "t", Body: []byte("oi")})

	select {
	case m := <-done:
		if string(m.Body) != "oi" {
			t.Fatalf("body errado: %q", string(m.Body))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler não foi chamado")
	}
}

func TestInProcess_RetriesUntilSuccess(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	done := make(chan struct{}, 1)

	q := NewInProcess(1, 16, 3, time.Millisecond, nil)
	q.Register("t", func(_ context.Context, _ Message) error {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n < 3 {
			return errors.New("falha transitória")
		}
		done <- struct{}{}
		return nil
	})
	q.Start(context.Background())

	_ = q.Publish(context.Background(), Message{Type: "t"})

	select {
	case <-done:
		mu.Lock()
		a := attempts
		mu.Unlock()
		if a != 3 {
			t.Fatalf("esperava 3 tentativas (2 falhas + sucesso), got %d", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("não convergiu no retry")
	}
}

func TestInProcess_DropsAfterMaxRetries(t *testing.T) {
	var mu sync.Mutex
	calls := 0

	q := NewInProcess(1, 16, 2, time.Millisecond, nil)
	q.Register("t", func(_ context.Context, _ Message) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return errors.New("sempre falha")
	})
	q.Start(context.Background())
	_ = q.Publish(context.Background(), Message{Type: "t"})

	// maxRetries=2 → 1 tentativa inicial + 2 retries = 3 chamadas, depois desiste.
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	got := calls
	mu.Unlock()
	if got != 3 {
		t.Fatalf("esperava 3 chamadas (inicial + 2 retries), got %d", got)
	}
}
