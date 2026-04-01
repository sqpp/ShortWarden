package httpapi

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxUserIDKey  ctxKey = "userID"
	ctxAPIKeyIDKey ctxKey = "apiKeyID"
)

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxUserIDKey, id)
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(ctxUserIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

func withAPIKeyID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxAPIKeyIDKey, id)
}

func apiKeyIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(ctxAPIKeyIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}

