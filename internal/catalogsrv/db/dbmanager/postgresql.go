// Package dbmanager provides functionality for managing the PostgreSQL database connection pool and executing queries.
package dbmanager

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/lib/pq"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/db/config"
)

// postgresConn represents a connection to the PostgreSQL database.
type postgresConn struct {
	conn             *sql.Conn
	cancel           context.CancelFunc
	scopes           map[string]string
	configuredScopes []string
	pool             *postgresPool
}

// postgresPool represents a pool of PostgreSQL database connections.
type postgresPool struct {
	configuredScopes []string
	connRequests     uint64
	connReturns      uint64
	db               *sql.DB
}

// validScopeNameRegex ensures scope names are valid PostgreSQL identifiers
var validScopeNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)

// formatSQLIdentifier formats a scope name for use in SQL using proper identifier quoting.
func formatSQLIdentifier(name string) string {
	return pq.QuoteIdentifier(name)
}

// NewPostgresqlDb creates a new PostgreSQL database connection pool with the given configured scopes.
// It returns a pointer to the PostgresPool and an error, if any.
func NewPostgresqlDb(configuredScopes []string) (ScopedDb, error) {
	for _, scope := range configuredScopes {
		if !validScopeNameRegex.MatchString(scope) {
			return nil, fmt.Errorf("invalid scope name: %s", scope)
		}
	}

	dsn := config.HatchCatalogDsn()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Error().Err(err).Msg("failed to open db")
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool settings
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	err = sqlDB.Ping()
	if err != nil {
		log.Error().Err(err).Msg("failed to ping db")
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &postgresPool{
		configuredScopes: configuredScopes,
		db:               sqlDB,
	}, nil
}

// Conn returns a new connection to the PostgreSQL database from the connection pool.
func (p *postgresPool) Conn(ctx context.Context) (ScopedConn, error) {
	ctx, cancel := context.WithCancel(ctx)

	conn, err := p.db.Conn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to obtain connection")
		cancel()
		return nil, fmt.Errorf("failed to obtain database connection: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			cancel()
			conn.Close()
			log.Error().Interface("panic", r).Msg("recovered from panic while setting up connection")
		}
	}()

	sessionParams := map[string]string{
		"lock_timeout":                        "5s",
		"statement_timeout":                   "5s",
		"idle_in_transaction_session_timeout": "5s",
	}

	for param, value := range sessionParams {
		// For SET commands, we need to properly quote both the parameter and value
		query := fmt.Sprintf("SET %s = %s", formatSQLIdentifier(param), pq.QuoteLiteral(value))
		_, err = conn.ExecContext(ctx, query)
		if err != nil {
			cancel()
			conn.Close()
			return nil, fmt.Errorf("failed to set %s: %w", param, err)
		}
	}

	h := &postgresConn{
		configuredScopes: p.configuredScopes,
		scopes:           make(map[string]string),
		cancel:           cancel,
		pool:             p,
		conn:             conn,
	}

	if err := h.DropScopes(ctx, p.configuredScopes); err != nil {
		cancel()
		conn.Close()
		return nil, fmt.Errorf("failed to initialize scopes: %w", err)
	}

	atomic.AddUint64(&p.connRequests, 1)
	return h, nil
}

// Stats returns the number of connection requests and returns made to the PostgreSQL database.
func (p *postgresPool) Stats() (requests, returns uint64) {
	return atomic.LoadUint64(&p.connRequests), atomic.LoadUint64(&p.connReturns)
}

// OpenConns returns the number of open connections in the pool.
func (p *postgresPool) OpenConns() int {
	return p.db.Stats().OpenConnections
}

// Close cleans up the scopes and returns the connection back to the pool.
func (h *postgresConn) Close(ctx context.Context) {
	if h.conn == nil {
		return
	}

	if err := h.DropAllScopes(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to drop all scopes during connection close")
	}

	if h.conn != nil {
		h.conn.Close()
	}
	if h.cancel != nil {
		h.cancel()
	}

	atomic.AddUint64(&h.pool.connReturns, 1)
}

// IsConfiguredScope checks if the given scope is configured in the PostgresConn.
func (h *postgresConn) IsConfiguredScope(scope string) bool {
	for _, s := range h.configuredScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// AddScopes adds the given scopes to the PostgresConn.
func (h *postgresConn) AddScopes(ctx context.Context, scopes map[string]string) error {
	if h.conn == nil {
		return fmt.Errorf("no active connection")
	}

	for scope := range scopes {
		if !validScopeNameRegex.MatchString(scope) {
			return fmt.Errorf("invalid scope name: %s", scope)
		}
	}

	tx, err := h.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for setting scopes: %w", err)
	}
	defer tx.Rollback()

	for scope, value := range scopes {
		if h.IsConfiguredScope(scope) {
			query := fmt.Sprintf("SET %s = %s", formatSQLIdentifier(scope), pq.QuoteLiteral(value))
			_, err := tx.ExecContext(ctx, query)
			if err != nil {
				return fmt.Errorf("failed to set scope %q: %w", scope, err)
			}
			h.scopes[scope] = value
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit scope changes: %w", err)
	}

	return nil
}

// AddScope adds a single scope to the PostgresConn.
func (h *postgresConn) AddScope(ctx context.Context, scope, value string) error {
	if h.conn == nil {
		return fmt.Errorf("no active connection")
	}

	if !validScopeNameRegex.MatchString(scope) {
		return fmt.Errorf("invalid scope name: %s", scope)
	}

	if h.IsConfiguredScope(scope) {
		query := fmt.Sprintf("SET %s = %s", formatSQLIdentifier(scope), pq.QuoteLiteral(value))
		_, err := h.conn.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to set scope %q: %w", scope, err)
		}
		h.scopes[scope] = value
	}

	return nil
}

// AuthorizedScopes returns the currently authorized scopes in the PostgresConn.
func (h *postgresConn) AuthorizedScopes() map[string]string {
	sc := make(map[string]string, len(h.scopes))
	for k, v := range h.scopes {
		sc[k] = v
	}
	return sc
}

// DropScopes drops the given scopes from the PostgresConn.
func (h *postgresConn) DropScopes(ctx context.Context, scopes []string) error {
	if h.conn == nil {
		return nil
	}

	for _, scope := range scopes {
		if !validScopeNameRegex.MatchString(scope) {
			return fmt.Errorf("invalid scope name: %s", scope)
		}
	}

	tx, err := h.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for dropping scopes: %w", err)
	}
	defer tx.Rollback()

	for _, scope := range scopes {
		query := fmt.Sprintf("RESET %s", formatSQLIdentifier(scope))
		_, err := tx.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to reset scope %q: %w", scope, err)
		}
		delete(h.scopes, scope)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit scope changes: %w", err)
	}

	return nil
}

// DropScope drops a single scope from the PostgresConn.
func (h *postgresConn) DropScope(ctx context.Context, scope string) error {
	if h.conn == nil {
		return nil
	}

	if !validScopeNameRegex.MatchString(scope) {
		return fmt.Errorf("invalid scope name: %s", scope)
	}

	query := fmt.Sprintf("RESET %s", formatSQLIdentifier(scope))
	_, err := h.conn.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to reset scope %q: %w", scope, err)
	}
	delete(h.scopes, scope)
	return nil
}

// DropAllScopes drops all the configured scopes from the PostgresConn.
func (h *postgresConn) DropAllScopes(ctx context.Context) error {
	return h.DropScopes(ctx, h.configuredScopes)
}

// Conn returns the underlying connection of the PostgresConn.
func (h *postgresConn) Conn() *sql.Conn {
	return h.conn
}
