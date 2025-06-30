// Package db provides database interfaces and implementations for the catalog service.
// It defines three main interfaces:
// - MetadataManager: Handles metadata operations like tenants, projects, catalogs, etc.
// - ObjectManager: Manages catalog objects and resources
// - ConnectionManager: Manages database connections and scopes
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dbmanager"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/db/postgresql"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// Database is an interface for the database connection. It wraps the underlying sql.Conn interface while
// adding the ability to manage scopes.
// The three interfaces are separately initialized to allow for wrapping each interface separately.
// This is particularly useful for caching. ObjectManager is a prime candidate for caching.

// MetadataManager handles all metadata operations in the catalog service.
// It manages tenants, projects, catalogs, variants, namespaces, views, and signing keys.
// All operations require a valid context and may return apperrors.Error for various failure cases.
type MetadataManager interface {
	//Tenant and Project
	CreateTenant(ctx context.Context, tenantID catcommon.TenantId) error
	GetTenant(ctx context.Context, tenantID catcommon.TenantId) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, tenantID catcommon.TenantId) error
	CreateProject(ctx context.Context, projectID catcommon.ProjectId) error
	GetProject(ctx context.Context, projectID catcommon.ProjectId) (*models.Project, error)
	DeleteProject(ctx context.Context, projectID catcommon.ProjectId) error

	// Catalog
	CreateCatalog(ctx context.Context, catalog *models.Catalog) apperrors.Error
	GetCatalogIDByName(ctx context.Context, catalogName string) (uuid.UUID, apperrors.Error)
	GetCatalogByID(ctx context.Context, catalogID uuid.UUID) (*models.Catalog, apperrors.Error)
	GetCatalogByName(ctx context.Context, name string) (*models.Catalog, apperrors.Error)
	ListCatalogs(ctx context.Context) ([]*models.Catalog, apperrors.Error)
	UpdateCatalog(ctx context.Context, catalog *models.Catalog) apperrors.Error
	DeleteCatalog(ctx context.Context, catalogID uuid.UUID, name string) apperrors.Error

	// Variant
	CreateVariant(ctx context.Context, variant *models.Variant) apperrors.Error
	GetVariant(ctx context.Context, catalogID uuid.UUID, variantID uuid.UUID, name string) (*models.Variant, apperrors.Error)
	GetVariantByID(ctx context.Context, variantID uuid.UUID) (*models.Variant, apperrors.Error)
	GetVariantIDFromName(ctx context.Context, catalogID uuid.UUID, name string) (uuid.UUID, apperrors.Error)
	ListVariantsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]models.VariantSummary, apperrors.Error)
	UpdateVariant(ctx context.Context, variantID uuid.UUID, name string, updatedVariant *models.Variant) apperrors.Error
	DeleteVariant(ctx context.Context, catalogID uuid.UUID, variantID uuid.UUID, name string) apperrors.Error
	GetMetadataNames(ctx context.Context, catalogID uuid.UUID, variantID uuid.UUID) (string, string, apperrors.Error)

	// Namespace
	CreateNamespace(ctx context.Context, ns *models.Namespace) apperrors.Error
	GetNamespace(ctx context.Context, name string, variantID uuid.UUID) (*models.Namespace, apperrors.Error)
	UpdateNamespace(ctx context.Context, ns *models.Namespace) apperrors.Error
	DeleteNamespace(ctx context.Context, name string, variantID uuid.UUID) apperrors.Error
	ListNamespacesByVariant(ctx context.Context, variantID uuid.UUID) ([]*models.Namespace, apperrors.Error)

	// View
	CreateView(ctx context.Context, view *models.View) apperrors.Error
	GetView(ctx context.Context, viewID uuid.UUID) (*models.View, apperrors.Error)
	GetViewByLabel(ctx context.Context, label string, catalogID uuid.UUID) (*models.View, apperrors.Error)
	UpdateView(ctx context.Context, view *models.View) apperrors.Error
	DeleteView(ctx context.Context, viewID uuid.UUID) apperrors.Error
	DeleteViewByLabel(ctx context.Context, label string, catalogID uuid.UUID) apperrors.Error
	ListViewsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]*models.View, apperrors.Error)

	// Tangent
	CreateTangent(ctx context.Context, tangent *models.Tangent) apperrors.Error
	GetTangent(ctx context.Context, id uuid.UUID) (*models.Tangent, apperrors.Error)
	UpdateTangent(ctx context.Context, tangent *models.Tangent) apperrors.Error
	DeleteTangent(ctx context.Context, id uuid.UUID) apperrors.Error
	ListTangents(ctx context.Context) ([]*models.Tangent, apperrors.Error)

	// ViewToken
	CreateViewToken(ctx context.Context, token *models.ViewToken) apperrors.Error
	GetViewToken(ctx context.Context, tokenID uuid.UUID) (*models.ViewToken, apperrors.Error)
	UpdateViewTokenExpiry(ctx context.Context, tokenID uuid.UUID, expireAt time.Time) apperrors.Error
	DeleteViewToken(ctx context.Context, tokenID uuid.UUID) apperrors.Error

	// SigningKey
	CreateSigningKey(ctx context.Context, key *models.SigningKey) apperrors.Error
	GetSigningKey(ctx context.Context, keyID uuid.UUID) (*models.SigningKey, apperrors.Error)
	GetActiveSigningKey(ctx context.Context) (*models.SigningKey, apperrors.Error)
	UpdateSigningKeyActive(ctx context.Context, keyID uuid.UUID, isActive bool) apperrors.Error
	DeleteSigningKey(ctx context.Context, keyID uuid.UUID) apperrors.Error

	// Session
	UpsertSession(ctx context.Context, session *models.Session) apperrors.Error
	GetSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, apperrors.Error)
	UpdateSessionStatus(ctx context.Context, sessionID uuid.UUID, statusSummary string, status json.RawMessage) apperrors.Error
	UpdateSessionEnd(ctx context.Context, sessionID uuid.UUID, statusSummary string, status json.RawMessage) apperrors.Error
	UpdateSessionInfo(ctx context.Context, sessionID uuid.UUID, info json.RawMessage) apperrors.Error
	DeleteSession(ctx context.Context, sessionID uuid.UUID) apperrors.Error
	ListSessionsByCatalog(ctx context.Context, catalogID uuid.UUID) ([]*models.Session, apperrors.Error)
}

// ObjectManager handles all object-related operations in the catalog service.
// It manages catalog objects, resources, and schema directories.
// All operations require a valid context and may return apperrors.Error for various failure cases.
type ObjectManager interface {
	// Catalog Object
	CreateCatalogObject(ctx context.Context, obj *models.CatalogObject) apperrors.Error
	GetCatalogObject(ctx context.Context, hash string) (*models.CatalogObject, apperrors.Error)
	DeleteCatalogObject(ctx context.Context, t catcommon.CatalogObjectType, hash string) apperrors.Error

	// Resources
	UpsertResource(ctx context.Context, rg *models.Resource, directoryID uuid.UUID) apperrors.Error
	GetResource(ctx context.Context, path string, variantID uuid.UUID, directoryID uuid.UUID) (*models.Resource, apperrors.Error)
	GetResourceObject(ctx context.Context, path string, directoryID uuid.UUID) (*models.CatalogObject, apperrors.Error)
	UpdateResource(ctx context.Context, rg *models.Resource, directoryID uuid.UUID) apperrors.Error
	DeleteResource(ctx context.Context, path string, directoryID uuid.UUID) (string, apperrors.Error)
	UpsertResourceObject(ctx context.Context, rg *models.Resource, obj *models.CatalogObject, directoryID uuid.UUID) apperrors.Error
	ListResources(ctx context.Context, directoryID uuid.UUID) ([]models.Resource, apperrors.Error)

	// Skillsets
	UpsertSkillSet(ctx context.Context, ss *models.SkillSet, directoryID uuid.UUID) apperrors.Error
	GetSkillSet(ctx context.Context, path string, variantID uuid.UUID, directoryID uuid.UUID) (*models.SkillSet, apperrors.Error)
	GetSkillSetObject(ctx context.Context, path string, directoryID uuid.UUID) (*models.CatalogObject, apperrors.Error)
	UpdateSkillSet(ctx context.Context, ss *models.SkillSet, directoryID uuid.UUID) apperrors.Error
	DeleteSkillSet(ctx context.Context, path string, directoryID uuid.UUID) (string, apperrors.Error)
	UpsertSkillSetObject(ctx context.Context, ss *models.SkillSet, obj *models.CatalogObject, directoryID uuid.UUID) apperrors.Error
	ListSkillSets(ctx context.Context, directoryID uuid.UUID) ([]models.SkillSet, apperrors.Error)

	// Schema Directory
	CreateSchemaDirectory(ctx context.Context, t catcommon.CatalogObjectType, dir *models.SchemaDirectory) apperrors.Error
	SetDirectory(ctx context.Context, t catcommon.CatalogObjectType, id uuid.UUID, dir []byte) apperrors.Error
	GetDirectory(ctx context.Context, t catcommon.CatalogObjectType, id uuid.UUID) ([]byte, apperrors.Error)
	GetSchemaDirectory(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID) (*models.SchemaDirectory, apperrors.Error)
	GetObjectRefByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (*models.ObjectRef, apperrors.Error)
	LoadObjectByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (*models.CatalogObject, apperrors.Error)
	AddOrUpdateObjectByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string, obj models.ObjectRef) apperrors.Error
	DeleteObjectByPath(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (catcommon.Hash, apperrors.Error)
	PathExists(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, path string) (bool, apperrors.Error)
	DeleteNamespaceObjects(ctx context.Context, t catcommon.CatalogObjectType, directoryID uuid.UUID, namespace string) ([]string, apperrors.Error)
}

// ConnectionManager handles database connection and scope management.
// It provides methods to add and remove scopes, which are used to filter data based on tenant and project.
// All operations require a valid context and may return error for connection-related issues.
type ConnectionManager interface {
	// Scope Management
	AddScopes(ctx context.Context, scopes map[string]string) error
	DropScopes(ctx context.Context, scopes []string) error
	AddScope(ctx context.Context, scope, value string) error
	DropScope(ctx context.Context, scope string) error
	DropAllScopes(ctx context.Context) error

	// Close the connection to the database.
	Close(ctx context.Context)
}

// Database interface combines all three managers into a single interface.
// This allows for a unified database access layer while maintaining separation of concerns.
type Database interface {
	MetadataManager
	ObjectManager
	ConnectionManager
}

// Scope constants define the available scopes for database operations
const (
	// Scope_TenantId is used to filter data by tenant
	Scope_TenantId string = "tansive.curr_tenantid"
	// Scope_ProjectId is used to filter data by project
	Scope_ProjectId string = "tansive.curr_projectid"
)

var configuredScopes = []string{
	Scope_TenantId,
	Scope_ProjectId,
}

var pool dbmanager.ScopedDb

// init initializes the database connection pool.
// It attempts to create a new scoped database connection and logs any errors.
func Init() {
	ctx := log.Logger.WithContext(context.Background())
	pg := dbmanager.NewScopedDb(ctx, "postgresql", configuredScopes)
	if pg == nil {
		panic("unable to create db pool")
	}
	pool = pg
}

// Conn returns a new database connection from the pool.
// Returns an error if the connection cannot be established.
func Conn(ctx context.Context) (dbmanager.ScopedConn, error) {
	if pool != nil {
		conn, err := pool.Conn(ctx)
		if err == nil {
			return conn, nil
		}
		log.Ctx(ctx).Error().Err(err).Msg("unable to get db connection")
		return nil, err
	}
	return nil, fmt.Errorf("database pool not initialized")
}

type ctxDbKeyType string

const ctxDbKey ctxDbKeyType = "TansiveCatalogDb"

// ConnCtx adds a database connection to the context.
// Returns an error if the connection cannot be established.
func ConnCtx(ctx context.Context) (context.Context, error) {
	conn, err := Conn(ctx)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, ctxDbKey, conn), nil
}

type tansiveCatalogDb struct {
	MetadataManager
	ObjectManager
	ConnectionManager
}

// DB returns a new database instance from the context.
// It expects a valid database connection in the context.
// Returns nil if no connection is found in the context.
func DB(ctx context.Context) Database {
	if conn, ok := ctx.Value(ctxDbKey).(dbmanager.ScopedConn); ok {
		mm, om, cm := postgresql.NewHatchCatalogDb(conn)
		return &tansiveCatalogDb{
			MetadataManager:   mm,
			ObjectManager:     om,
			ConnectionManager: cm,
		}
	}
	log.Ctx(ctx).Error().Msg("unable to get db connection from context")
	return nil
}
