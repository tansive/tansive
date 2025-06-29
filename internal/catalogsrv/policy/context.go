package policy

import (
	"context"
)

type ctxKeyType string

var (
	ViewDefinitionContextKey ctxKeyType = "viewDefinition"
)

func WithViewDefinition(ctx context.Context, viewDefinition *ViewDefinition) context.Context {
	return context.WithValue(ctx, ViewDefinitionContextKey, viewDefinition)
}

func GetViewDefinition(ctx context.Context) *ViewDefinition {
	v, ok := ctx.Value(ViewDefinitionContextKey).(*ViewDefinition)
	if !ok {
		return nil
	}
	return v
}
