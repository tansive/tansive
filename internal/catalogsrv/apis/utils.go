package apis

import (
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tidwall/gjson"
)

// processPath normalizes a path and returns the object name and path.
// It handles edge cases like root paths and ensures consistent path formatting.
func processPath(p string) (objectName, objectPath string) {
	objectName = path.Base(p)
	if objectName == "/" || objectName == "." {
		objectName = ""
	}
	objectPath = path.Dir(p)
	if objectPath == "." {
		objectPath = "/"
	}
	objectPath = path.Clean("/" + objectPath)
	return
}

// hydrateRequestContext processes an HTTP request to extract and structure catalog-related context information.
func hydrateRequestContext(r *http.Request) (interfaces.RequestContext, error) {
	if r == nil {
		return interfaces.RequestContext{}, httpx.ErrInvalidRequest("request cannot be nil")
	}

	ctx := r.Context()
	viewName := chi.URLParam(r, "viewName")
	kindName := getResourceNameFromPath(r)

	n := interfaces.RequestContext{
		QueryParams: r.URL.Query(),
	}

	catalogCtx := catcommon.GetCatalogContext(ctx)
	if catalogCtx == nil {
		log.Ctx(ctx).Error().Msg("no catalog context found")
		return n, httpx.ErrInvalidRequest("missing catalog context")
	}

	n.Catalog = catalogCtx.Catalog
	n.CatalogID = catalogCtx.CatalogID
	n.Variant = catalogCtx.Variant
	n.VariantID = catalogCtx.VariantID
	n.Namespace = catalogCtx.Namespace

	// Handle view name
	if viewName != "" {
		n.ObjectName = viewName
	}

	// Process resource paths
	if kindName == catcommon.KindNameResources {
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/"+catcommon.KindNameResources+"/definition"):
			resourcePath := strings.TrimPrefix(path, "/"+catcommon.KindNameResources+"/definition")
			resourcePath = strings.TrimPrefix(resourcePath, "/")
			n.ObjectName, n.ObjectPath = processPath(resourcePath)
			n.ObjectType = catcommon.CatalogObjectTypeResource
			n.ObjectProperty = catcommon.ResourcePropertyDefinition
		default:
			resourceValue := strings.TrimPrefix(path, "/"+catcommon.KindNameResources)
			resourceValue = strings.TrimPrefix(resourceValue, "/")
			n.ObjectName, n.ObjectPath = processPath(resourceValue)
			n.ObjectType = catcommon.CatalogObjectTypeResource
			n.ObjectProperty = catcommon.ResourcePropertyValue
		}
	}

	// Process skillset paths
	if kindName == catcommon.KindNameSkillsets {
		skillsetPath := strings.TrimPrefix(r.URL.Path, "/"+catcommon.KindNameSkillsets)
		skillsetPath = strings.TrimPrefix(skillsetPath, "/")
		n.ObjectName, n.ObjectPath = processPath(skillsetPath)
		n.ObjectType = catcommon.CatalogObjectTypeSkillset
	}

	return n, nil
}

func getResourceKind(r *http.Request) string {
	return catcommon.KindFromKindName(getResourceNameFromPath(r))
}

func getResourceNameFromPath(r *http.Request) string {
	path := strings.Trim(r.URL.Path, "/")
	segments := strings.Split(path, "/")
	var resourceName string
	if len(segments) > 0 {
		resourceName = segments[0]
	}
	return resourceName
}

func validateRequest(reqJSON []byte, kind string) error {
	if !gjson.ValidBytes(reqJSON) {
		return httpx.ErrInvalidRequest("unable to parse request")
	}
	if kind == catcommon.ResourceKind {
		return nil
	}
	result := gjson.GetBytes(reqJSON, "kind")
	if !result.Exists() {
		return httpx.ErrInvalidRequest("missing kind")
	}
	if result.String() != kind {
		return httpx.ErrInvalidRequest("invalid kind")
	}
	return nil
}
