// Package catcommon provides context management utilities for the catalog service.
// It includes functionality for managing tenant, project, catalog, and user contexts.
package catcommon

import (
	"context"

	"github.com/tansive/tansive/internal/common/uuid"
)

// ctxKeyType represents the type for all context keys
type ctxKeyType string

// Context keys for different types of data
const (
	// Catalog related keys
	ctxCatalogContextKey ctxKeyType = "CatalogContext"
	ctxTenantIdKey       ctxKeyType = "CatalogTenantId"
	ctxProjectIdKey      ctxKeyType = "CatalogProjectId"
	ctxTestContextKey    ctxKeyType = "CatalogTestContext"
)

type SubjectType string

const (
	SubjectTypeUser    SubjectType = "user"
	SubjectTypeSession SubjectType = "session"
	SubjectTypeSystem  SubjectType = "system"
	SubjectTypeService SubjectType = "service"
)

// CatalogContext represents the complete context for catalog operations.
// It contains all necessary information about the catalog, variant, and user.
type CatalogContext struct {
	// CatalogID is the unique identifier for the catalog
	CatalogID uuid.UUID
	// VariantID is the unique identifier for the variant
	VariantID uuid.UUID
	// Namespace is the namespace for the catalog
	Namespace string
	// Catalog is the name of the catalog
	Catalog string
	// Variant is the name of the variant
	Variant string
	// UserContext contains information about the authenticated user
	UserContext *UserContext
	// SessionContext contains information about the session
	SessionContext *SessionContext
	// Subject is the type of principal that is acting on the catalog
	Subject SubjectType
}

// UserContext represents the context of an authenticated user in the system.
// It contains information about the user's identity and permissions.
type UserContext struct {
	// UserID is the unique identifier for the user
	UserID string
}

// SessionContext represents the context of a session in the system.
// It contains information about the session's identity and permissions.
type SessionContext struct {
	// SessionID is the unique identifier for the session
	SessionID uuid.UUID
}

// WithTenantID sets the tenant ID in the provided context.
func WithTenantID(ctx context.Context, tenantId TenantId) context.Context {
	return context.WithValue(ctx, ctxTenantIdKey, tenantId)
}

// GetTenantID retrieves the tenant ID from the provided context.
func GetTenantID(ctx context.Context) TenantId {
	if tenantId, ok := ctx.Value(ctxTenantIdKey).(TenantId); ok {
		return tenantId
	}
	return ""
}

// WithProjectID sets the project ID in the provided context.
func WithProjectID(ctx context.Context, projectId ProjectId) context.Context {
	return context.WithValue(ctx, ctxProjectIdKey, projectId)
}

// GetProjectID retrieves the project ID from the provided context.
func GetProjectID(ctx context.Context) ProjectId {
	if projectId, ok := ctx.Value(ctxProjectIdKey).(ProjectId); ok {
		return projectId
	}
	return ""
}

// WithCatalogContext sets the catalog context in the provided context.
func WithCatalogContext(ctx context.Context, catalogContext *CatalogContext) context.Context {
	return context.WithValue(ctx, ctxCatalogContextKey, catalogContext)
}

// GetCatalogContext retrieves the catalog context from the provided context.
func GetCatalogContext(ctx context.Context) *CatalogContext {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext
	}
	return nil
}

// WithCatalogID sets the catalog ID in the provided context.
func WithCatalogID(ctx context.Context, catalogId uuid.UUID) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	currContext.CatalogID = catalogId
	return WithCatalogContext(ctx, currContext)
}

// WithVariantID sets the variant ID in the provided context.
func WithVariantID(ctx context.Context, variantId uuid.UUID) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	currContext.VariantID = variantId
	return WithCatalogContext(ctx, currContext)
}

// WithNamespace sets the namespace in the provided context.
func WithNamespace(ctx context.Context, namespace string) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	currContext.Namespace = namespace
	return WithCatalogContext(ctx, currContext)
}

// WithCatalog sets the catalog in the provided context.
func WithCatalog(ctx context.Context, catalog string) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	currContext.Catalog = catalog
	return WithCatalogContext(ctx, currContext)
}

// WithVariant sets the variant in the provided context.
func WithVariant(ctx context.Context, variant string) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	currContext.Variant = variant
	return WithCatalogContext(ctx, currContext)
}

func WithSessionID(ctx context.Context, sessionID uuid.UUID) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	if currContext.SessionContext == nil {
		currContext.SessionContext = &SessionContext{}
	}
	currContext.SessionContext.SessionID = sessionID
	return WithCatalogContext(ctx, currContext)
}

func WithSessionContext(ctx context.Context, sessionContext *SessionContext) context.Context {
	currContext := GetCatalogContext(ctx)
	if currContext == nil {
		currContext = &CatalogContext{}
	}
	currContext.SessionContext = sessionContext
	return WithCatalogContext(ctx, currContext)
}

// GetCatalogID retrieves the catalog ID from the provided context.
func GetCatalogID(ctx context.Context) uuid.UUID {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.CatalogID
	}
	return uuid.Nil
}

// GetVariantID retrieves the variant ID from the provided context.
func GetVariantID(ctx context.Context) uuid.UUID {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.VariantID
	}
	return uuid.Nil
}

// GetNamespace retrieves the namespace from the provided context.
func GetNamespace(ctx context.Context) string {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.Namespace
	}
	return ""
}

// GetCatalog retrieves the catalog from the provided context.
func GetCatalog(ctx context.Context) string {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.Catalog
	}
	return ""
}

// GetVariant retrieves the variant from the provided context.
func GetVariant(ctx context.Context) string {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.Variant
	}
	return ""
}

// GetUserContext retrieves the user context from the provided context.
func GetUserContext(ctx context.Context) *UserContext {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.UserContext
	}
	return nil
}

func GetUserID(ctx context.Context) string {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		if catalogContext.UserContext != nil {
			return catalogContext.UserContext.UserID
		}
	}
	return ""
}

func GetSessionContext(ctx context.Context) *SessionContext {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.SessionContext
	}
	return nil
}

func GetSessionID(ctx context.Context) uuid.UUID {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		if catalogContext.SessionContext != nil {
			return catalogContext.SessionContext.SessionID
		}
	}
	return uuid.Nil
}

func GetSubjectType(ctx context.Context) SubjectType {
	if catalogContext, ok := ctx.Value(ctxCatalogContextKey).(*CatalogContext); ok {
		return catalogContext.Subject
	}
	return SubjectType("")
}

// WithTestContext sets the test context in the provided context.
func WithTestContext(ctx context.Context, isTest bool) context.Context {
	return context.WithValue(ctx, ctxTestContextKey, isTest)
}

// GetTestContext retrieves the test context from the provided context.
func GetTestContext(ctx context.Context) bool {
	if testContext, ok := ctx.Value(ctxTestContextKey).(bool); ok {
		return testContext
	}
	return false
}
