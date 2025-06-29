package apis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tidwall/gjson"
)

func withContext(r *http.Request) (*http.Request, error) {
	ctx := r.Context()

	// Get initial project ID
	projectID := getProjectIDFromRequest(r)

	catalogCtx := catcommon.GetCatalogContext(ctx)
	if catalogCtx == nil {
		catalogCtx = &catcommon.CatalogContext{}
	}

	// Load metadata from various sources
	catalogCtx = loadMetadataFromParam(r, catalogCtx)
	catalogCtx = loadMetadataFromQuery(r, catalogCtx)

	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		var err error
		catalogCtx, err = loadMetadataFromBody(r, catalogCtx)
		if err != nil {
			return r, fmt.Errorf("failed to load metadata from body: %w", err)
		}
	}

	// Try to resolve project ID and catalog info if needed
	if projectID == "" {
		var err error
		projectID, err = resolveProjectIDFromCatalog(ctx, catalogCtx)
		if err != nil {
			return r, fmt.Errorf("failed to resolve project ID from catalog: %w", err)
		}
	}

	// If we are in single user mode, use the default project ID, otherwise return an error
	if projectID == "" {
		if config.Config().SingleUserMode {
			projectID = catcommon.ProjectId(config.Config().DefaultProjectID)
		} else {
			return r, fmt.Errorf("project ID is required")
		}
	}

	ctx = catcommon.WithProjectID(ctx, projectID)

	if err := resolveCatalogInfo(ctx, catalogCtx); err != nil {
		return r, fmt.Errorf("failed to resolve catalog info: %w", err)
	}

	// Resolve variant information
	if err := resolveVariantInfo(ctx, catalogCtx); err != nil {
		return r, fmt.Errorf("failed to resolve variant info: %w", err)
	}

	// Set the final context
	r = r.WithContext(catcommon.WithCatalogContext(ctx, catalogCtx))
	return r, nil
}

func loadMetadataFromParam(r *http.Request, catalogCtx *catcommon.CatalogContext) *catcommon.CatalogContext {
	if catalogCtx == nil {
		return nil
	}

	catalogName := chi.URLParam(r, "catalogName")
	variantName := chi.URLParam(r, "variantName")
	namespace := chi.URLParam(r, "namespaceName")

	if catalogName != "" {
		catalogCtx.Catalog = catalogName
	}
	if variantName != "" {
		catalogCtx.Variant = variantName
	}
	if namespace != "" {
		catalogCtx.Namespace = namespace
	}

	return catalogCtx
}

func loadMetadataFromQuery(r *http.Request, catalogCtx *catcommon.CatalogContext) *catcommon.CatalogContext {
	if catalogCtx == nil {
		return nil
	}

	urlValues := r.URL.Query()

	if catalogCtx.CatalogID == uuid.Nil && catalogCtx.Catalog == "" {
		catalogID := getURLValue(urlValues, "catalog_id")
		if catalogID != "" {
			catalogUUID, err := uuid.Parse(catalogID)
			if err == nil {
				catalogCtx.CatalogID = catalogUUID
			}
		} else {
			catalog := getURLValue(urlValues, "catalog")
			if catalog != "" {
				catalogCtx.Catalog = catalog
			}
		}
	}

	if catalogCtx.VariantID == uuid.Nil && catalogCtx.Variant == "" {
		variantID := getURLValue(urlValues, "variant_id")
		if variantID != "" {
			variantUUID, err := uuid.Parse(variantID)
			if err == nil {
				catalogCtx.VariantID = variantUUID
			}
		} else {
			variant := getURLValue(urlValues, "variant")
			if variant != "" {
				catalogCtx.Variant = variant
			}
		}
	}

	if catalogCtx.Namespace == "" {
		namespace := getURLValue(urlValues, "namespace")
		if namespace != "" {
			catalogCtx.Namespace = namespace
		}
	}

	return catalogCtx
}

func loadMetadataFromBody(r *http.Request, catalogCtx *catcommon.CatalogContext) (*catcommon.CatalogContext, error) {
	if catalogCtx == nil {
		return nil, nil
	}
	if r.Body == nil {
		return catalogCtx, nil
	}
	w := httptest.NewRecorder() // we need a fake response writer
	r.Body = http.MaxBytesReader(w, r.Body, config.Config().MaxRequestBodySize)
	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		return nil, err
	}
	// Restore body for downstream handlers using the buffered content
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Parse metadata
	if catalogCtx.VariantID == uuid.Nil && catalogCtx.Variant == "" {
		if result := gjson.GetBytes(body, "metadata.variant"); result.Exists() {
			catalogCtx.Variant = result.String()
		}
	}
	if catalogCtx.Namespace == "" {
		if result := gjson.GetBytes(body, "metadata.namespace"); result.Exists() {
			catalogCtx.Namespace = result.String()
		}
	}

	return catalogCtx, nil
}

var urlKeyShorthand = map[string]string{
	"catalog_id": "c_id",
	"catalog":    "c",
	"variant_id": "v_id",
	"variant":    "v",
	"namespace":  "n",
}

// getURLValue retrieves a value from URL values, checking both full and shorthand keys
func getURLValue(values url.Values, key string) string {
	value := values.Get(key)
	if value == "" {
		if shorthand, ok := urlKeyShorthand[key]; ok {
			value = values.Get(shorthand)
		}
	}
	return value
}

func getProjectIDFromRequest(r *http.Request) catcommon.ProjectId {
	projectID := r.URL.Query().Get("project")
	if projectID != "" {
		return catcommon.ProjectId(projectID)
	}
	return ""
}

func resolveProjectIDFromCatalog(ctx context.Context, catalogCtx *catcommon.CatalogContext) (catcommon.ProjectId, error) {
	if catalogCtx.CatalogID != uuid.Nil {
		catalog, err := db.DB(ctx).GetCatalogByID(ctx, catalogCtx.CatalogID)
		if err != nil {
			return "", fmt.Errorf("failed to get catalog by ID: %w", err)
		}
		catalogCtx.Catalog = catalog.Name
		return catalog.ProjectID, nil
	}
	return "", nil
}

// resolveCatalogInfo resolves catalog information using either name or ID
// Only called when we already have a project ID
func resolveCatalogInfo(ctx context.Context, catalogCtx *catcommon.CatalogContext) error {
	if catalogCtx.Catalog != "" && catalogCtx.CatalogID == uuid.Nil {
		catalog, err := db.DB(ctx).GetCatalogByName(ctx, catalogCtx.Catalog)
		if err != nil {
			return fmt.Errorf("failed to get catalog by name: %w", err)
		}
		catalogCtx.CatalogID = catalog.CatalogID
	} else if catalogCtx.CatalogID != uuid.Nil && catalogCtx.Catalog == "" {
		catalog, err := db.DB(ctx).GetCatalogByID(ctx, catalogCtx.CatalogID)
		if err != nil {
			return fmt.Errorf("failed to get catalog by ID: %w", err)
		}
		catalogCtx.Catalog = catalog.Name
	}
	return nil
}

// resolveVariantInfo resolves variant information using either name or ID
func resolveVariantInfo(ctx context.Context, catalogCtx *catcommon.CatalogContext) error {
	if catalogCtx == nil {
		return nil
	}

	// If we have neither variant ID nor name, nothing to resolve
	if catalogCtx.VariantID == uuid.Nil && catalogCtx.Variant == "" {
		return nil
	}

	// If we have variant ID but no name, fetch the name
	if catalogCtx.VariantID != uuid.Nil && catalogCtx.Variant == "" {
		variant, err := db.DB(ctx).GetVariant(ctx, catalogCtx.CatalogID, catalogCtx.VariantID, "")
		if err != nil {
			return fmt.Errorf("failed to get variant by ID: %w", err)
		}
		catalogCtx.Variant = variant.Name
		return nil
	}

	// If we have variant name but no ID, fetch the ID
	if catalogCtx.VariantID == uuid.Nil && catalogCtx.Variant != "" {
		variant, err := db.DB(ctx).GetVariant(ctx, catalogCtx.CatalogID, uuid.Nil, catalogCtx.Variant)
		if err != nil {
			return fmt.Errorf("failed to get variant by name: %w", err)
		}
		catalogCtx.VariantID = variant.VariantID
		return nil
	}

	return nil
}
