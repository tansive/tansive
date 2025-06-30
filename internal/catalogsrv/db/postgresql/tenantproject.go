package postgresql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"

	"github.com/rs/zerolog/log"
)

// CreateProjectAndTenant creates a new project and tenant in the database.
func (mm *metadataManager) CreateProjectAndTenant(ctx context.Context, projectID catcommon.ProjectId, tenantID catcommon.TenantId) error {
	// Create the tenant
	err := mm.CreateTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, dberror.ErrAlreadyExists) {
			return nil
		}
		return err
	}
	// Create the project
	err = mm.CreateProject(ctx, projectID)
	if err != nil {
		if errors.Is(err, dberror.ErrAlreadyExists) {
			return nil
		}
		return err
	}
	return nil
}

// CreateTenant inserts a new tenant into the database.
// If the tenant already exists, it does nothing.
func (mm *metadataManager) CreateTenant(ctx context.Context, tenantID catcommon.TenantId) error {
	query := `
		INSERT INTO tenants (tenant_id)
		VALUES ($1)
		ON CONFLICT (tenant_id) DO NOTHING
		RETURNING tenant_id;
	`

	// Execute the query directly using mm.conn().QueryRowContext
	row := mm.conn().QueryRowContext(ctx, query, string(tenantID))
	var insertedTenantID string
	err := row.Scan(&insertedTenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("tenant_id", string(tenantID)).Msg("tenant already exists")
			return dberror.ErrAlreadyExists.Msg("tenant already exists")
		}
		log.Ctx(ctx).Error().Str("tenant_id", string(tenantID)).Msg("failed to insert tenant")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

// GetTenant retrieves a tenant from the database.
func (mm *metadataManager) GetTenant(ctx context.Context, tenantID catcommon.TenantId) (*models.Tenant, error) {
	query := `
		SELECT tenant_id
		FROM tenants
		WHERE tenant_id = $1;
	`

	// Execute the query directly using mm.conn().QueryRowContext
	row := mm.conn().QueryRowContext(ctx, query, string(tenantID))

	var tenant models.Tenant
	err := row.Scan(&tenant.TenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("tenant_id", string(tenantID)).Msg("tenant not found")
			return nil, dberror.ErrNotFound.Msg("tenant not found")
		}
		log.Ctx(ctx).Error().Str("tenant_id", string(tenantID)).Msg("failed to retrieve tenant")
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &tenant, nil
}

// DeleteTenant deletes a tenant from the database.
func (mm *metadataManager) DeleteTenant(ctx context.Context, tenantID catcommon.TenantId) error {
	query := `
		DELETE FROM tenants
		WHERE tenant_id = $1;
	`
	_, err := mm.conn().ExecContext(ctx, query, string(tenantID))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("tenant_id", string(tenantID)).Msg("failed to delete tenant")
		return dberror.ErrDatabase.Err(err)
	}
	return nil
}

// CreateProject inserts a new project into the database.
func (mm *metadataManager) CreateProject(ctx context.Context, projectID catcommon.ProjectId) error {
	tenantID := catcommon.GetTenantID(ctx)

	// Validate tenantID to ensure it is not empty
	if tenantID == "" {
		log.Ctx(ctx).Error().Msg("tenant ID is missing from context")
		return dberror.ErrInvalidInput.Msg("tenant ID is required")
	}

	query := `
		INSERT INTO projects (project_id, tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (tenant_id, project_id) DO NOTHING
		RETURNING project_id;
	`

	// Execute the query directly using mm.conn().QueryRowContext
	row := mm.conn().QueryRowContext(ctx, query, string(projectID), string(tenantID))
	var insertedProjectID string
	err := row.Scan(&insertedProjectID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().Str("project_id", string(projectID)).Msg("project already exists")
			return dberror.ErrAlreadyExists.Msg("project already exists")
		}
		log.Ctx(ctx).Error().Str("project_id", string(projectID)).Msg("failed to insert project")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}

// GetProject retrieves a project from the database.
func (mm *metadataManager) GetProject(ctx context.Context, projectID catcommon.ProjectId) (*models.Project, error) {
	tenantID := catcommon.GetTenantID(ctx)

	// Validate tenantID to ensure it is not empty
	if tenantID == "" {
		log.Ctx(ctx).Error().Msg("tenant ID is missing from context")
		return nil, dberror.ErrInvalidInput.Msg("tenant ID is required")
	}

	query := `
		SELECT project_id, tenant_id
		FROM projects
		WHERE tenant_id = $1 AND project_id = $2;
	`

	// Query the project data
	row := mm.conn().QueryRowContext(ctx, query, string(tenantID), string(projectID))

	var project models.Project
	err := row.Scan(&project.ProjectID, &project.TenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Ctx(ctx).Info().
				Str("project_id", string(projectID)).
				Msg("project not found")
			return nil, dberror.ErrNotFound.Msg("project not found")
		}
		log.Ctx(ctx).Error().
			Str("project_id", string(projectID)).
			Msg("failed to retrieve project")
		return nil, dberror.ErrDatabase.Err(err)
	}

	return &project, nil
}

// DeleteProject deletes a project from the database. If the project does not exist, it does nothing.
func (mm *metadataManager) DeleteProject(ctx context.Context, projectID catcommon.ProjectId) error {
	tenantID := catcommon.GetTenantID(ctx)

	// Validate tenantID to ensure it is not empty
	if tenantID == "" {
		log.Ctx(ctx).Error().Msg("tenant ID is missing from context")
		return dberror.ErrInvalidInput.Msg("tenant ID is required")
	}

	query := `
		DELETE FROM projects
		WHERE tenant_id = $1 AND project_id = $2;
	`
	_, err := mm.conn().ExecContext(ctx, query, string(tenantID), string(projectID))
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("project_id", string(projectID)).
			Msg("failed to delete project")
		return dberror.ErrDatabase.Err(err)
	}

	return nil
}
