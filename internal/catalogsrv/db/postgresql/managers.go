package postgresql

import (
	"context"
	"database/sql"

	"github.com/tansive/tansive/internal/catalogsrv/db/dbmanager"
)

// Metadata Manager
type metadataManager struct {
	c dbmanager.ScopedConn
}

func (mm *metadataManager) conn() *sql.Conn {
	return mm.c.Conn()
}

func newMetadataManager(c dbmanager.ScopedConn) *metadataManager {
	return &metadataManager{c: c}
}

// Object Manager
type objectManager struct {
	c dbmanager.ScopedConn
	m *metadataManager
}

func (om *objectManager) conn() *sql.Conn {
	return om.c.Conn()
}

func newObjectManager(c dbmanager.ScopedConn) *objectManager {
	return &objectManager{c: c}
}

// Connection Manager
type connectionManager struct {
	c dbmanager.ScopedConn
}

func newConnectionManager(c dbmanager.ScopedConn) *connectionManager {
	return &connectionManager{c: c}
}

func (cm *connectionManager) AddScopes(ctx context.Context, scopes map[string]string) error {
	return cm.c.AddScopes(ctx, scopes)
}

func (cm *connectionManager) DropScopes(ctx context.Context, scopes []string) error {
	return cm.c.DropScopes(ctx, scopes)
}

func (cm *connectionManager) AddScope(ctx context.Context, scope, value string) error {
	return cm.c.AddScope(ctx, scope, value)
}

func (cm *connectionManager) DropScope(ctx context.Context, scope string) error {
	return cm.c.DropScope(ctx, scope)
}

func (cm *connectionManager) DropAllScopes(ctx context.Context) error {
	return cm.c.DropAllScopes(ctx)
}

func (cm *connectionManager) Close(ctx context.Context) {
	cm.c.Close(ctx)
}
