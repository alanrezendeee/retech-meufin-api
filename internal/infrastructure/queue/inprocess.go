package queue

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// InProcess é uma fila em memória com pool de workers, retry com backoff
// exponencial e enfileiramento não-bloqueante. Não é durável por si só — a
// durabilidade vem do registro persistente do trabalho (job no banco) somado ao
// sweeper que reenfileira o que ficou pendente após um restart.
//
// Substituível por um adaptador RabbitMQ que implemente Publisher: a lógica de
// negócio depende só da interface, não desta struct.
type InProcess struct {
	ch         chan task
	handlers   map[string]HandlerFunc
	workers    int
	maxRetries int
	backoff    time.Duration
	log        *slog.Logger

	mu      sync.Mutex
	started bool
}

type task struct {
	msg     Message
	attempt int
}

// NewInProcess cria a fila. workers = concorrência; buffer = tamanho do canal;
// maxRetries = tentativas extras por mensagem; backoff = base do atraso.
func NewInProcess(workers, buffer, maxRetries int, backoff time.Duration, log *slog.Logger) *InProcess {
	if workers <= 0 {
		workers = 4
	}
	if buffer <= 0 {
		buffer = 256
	}
	if backoff <= 0 {
		backoff = 3 * time.Second
	}
	return &InProcess{
		ch:         make(chan task, buffer),
		handlers:   make(map[string]HandlerFunc),
		workers:    workers,
		maxRetries: maxRetries,
		backoff:    backoff,
		log:        log,
	}
}

// Register associa um handler a um Type. Deve ser chamado antes de Start.
func (q *InProcess) Register(msgType string, h HandlerFunc) {
	q.handlers[msgType] = h
}

// Publish enfileira uma mensagem sem bloquear o chamador. Se o buffer estiver
// cheio, o envio é feito em background para não perder a mensagem (backstop; sob
// carga sustentada o broker real assume esse papel).
func (q *InProcess) Publish(_ context.Context, msg Message) error {
	q.enqueue(task{msg: msg, attempt: 0})
	return nil
}

func (q *InProcess) enqueue(t task) {
	select {
	case q.ch <- t:
	default:
		go func() { q.ch <- t }()
	}
}

// Start sobe o pool de workers. Idempotente. Encerra quando ctx é cancelado.
func (q *InProcess) Start(ctx context.Context) {
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return
	}
	q.started = true
	q.mu.Unlock()

	for i := 0; i < q.workers; i++ {
		go q.worker(ctx)
	}
	if q.log != nil {
		q.log.Info("🧵 fila in-process iniciada", slog.Int("workers", q.workers))
	}
}

func (q *InProcess) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-q.ch:
			q.process(ctx, t)
		}
	}
}

func (q *InProcess) process(ctx context.Context, t task) {
	defer func() {
		// Um handler em pânico não pode derrubar o worker.
		if r := recover(); r != nil && q.log != nil {
			q.log.Error("handler de fila entrou em pânico",
				slog.String("type", t.msg.Type), slog.Any("panic", r))
		}
	}()

	h := q.handlers[t.msg.Type]
	if h == nil {
		if q.log != nil {
			q.log.Warn("mensagem sem handler registrado", slog.String("type", t.msg.Type))
		}
		return
	}

	if err := h(ctx, t.msg); err != nil {
		if t.attempt < q.maxRetries {
			delay := q.backoff * time.Duration(int64(1)<<t.attempt) // backoff exponencial
			if q.log != nil {
				q.log.Warn("retry de mensagem",
					slog.String("type", t.msg.Type),
					slog.Int("attempt", t.attempt+1),
					slog.Duration("delay", delay),
					slog.String("error", err.Error()))
			}
			next := task{msg: t.msg, attempt: t.attempt + 1}
			time.AfterFunc(delay, func() { q.enqueue(next) })
			return
		}
		if q.log != nil {
			q.log.Error("mensagem descartada após esgotar retries",
				slog.String("type", t.msg.Type),
				slog.Int("attempts", t.attempt+1),
				slog.String("error", err.Error()))
		}
	}
}
