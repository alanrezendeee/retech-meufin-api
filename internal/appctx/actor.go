// Package appctx carrega valores transversais no context da requisição —
// hoje, o ator autenticado. Vive fora das camadas para que tanto o HTTP
// (que grava) quanto a aplicação (que lê) dependam só dele, sem inverter a
// direção das dependências.
package appctx

import (
	"context"

	"github.com/google/uuid"
)

type actorKey struct{}

// WithActor grava o usuário autenticado no context.
func WithActor(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, actorKey{}, userID)
}

// ActorFromContext devolve o usuário autenticado, se presente. Nulo em
// rotinas automáticas (extensor de recorrências, workers).
func ActorFromContext(ctx context.Context) *uuid.UUID {
	if v, ok := ctx.Value(actorKey{}).(uuid.UUID); ok {
		return &v
	}
	return nil
}
