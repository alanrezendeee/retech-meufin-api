// Package queue define a porta de fila de mensagens do sistema e uma
// implementação in-process.
//
// A abstração é deliberadamente message-oriented (Type + Body) para que a troca
// futura por um broker (RabbitMQ) seja apenas um novo adaptador que implemente
// Publisher e entregue mensagens aos handlers registrados — sem tocar na lógica
// de negócio. Mapeamento pretendido para RabbitMQ:
//
//	Message.Type  → routing key (bind de fila por tipo)
//	Message.Body  → corpo da mensagem (JSON)
//	Publish       → basic.publish no exchange
//	Register/Start→ consumidor (basic.consume) despachando por routing key
//	retry/backoff → dead-letter exchange + TTL (ou plugin de delayed message)
package queue

import "context"

// Message é uma mensagem publicada na fila.
type Message struct {
	// Type roteia a mensagem para o handler correspondente (routing key).
	Type string
	// Body é o payload serializado (JSON).
	Body []byte
}

// Publisher publica mensagens na fila. Implementações: InProcess (atual) e,
// futuramente, um adaptador RabbitMQ.
type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

// HandlerFunc processa uma mensagem de um dado Type. Retornar erro agenda um
// retry (com backoff) até o limite configurado; esgotado, a mensagem é
// descartada — o registro durável do trabalho (ex.: a linha do job no banco)
// permanece como fonte de verdade e é reprocessado pelo sweeper.
type HandlerFunc func(ctx context.Context, msg Message) error
