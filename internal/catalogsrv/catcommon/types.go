package catcommon

import (
	"github.com/tansive/tansive/internal/common/uuid"
)

type TenantId string
type ProjectId string
type CatalogId uuid.UUID
type VariantId uuid.UUID
type Hash string
type RunnerID string

const DefaultVariant = "default"
const InitialVersionLabel = "init"
const DefaultNamespace = "--root--"
const DefaultAdminViewLabel = "_default-admin-view"

func (u CatalogId) String() string {
	return uuid.UUID(u).String()
}

func (u CatalogId) IsNil() bool {
	return u == CatalogId(uuid.Nil)
}

const (
	CatalogKind   = "Catalog"
	VariantKind   = "Variant"
	NamespaceKind = "Namespace"
	ResourceKind  = "Resource"
	SkillSetKind  = "SkillSet"
	ViewKind      = "View"
	InvalidKind   = "InvalidKind"
)

const (
	KindNameCatalogs   = "catalogs"
	KindNameVariants   = "variants"
	KindNameNamespaces = "namespaces"
	KindNameViews      = "views"
	KindNameResources  = "resources"
	KindNameSkillsets  = "skillsets"
)

func ValidKindNames() []string {
	return []string{
		KindNameCatalogs,
		KindNameVariants,
		KindNameNamespaces,
		KindNameViews,
		KindNameResources,
		KindNameSkillsets,
	}
}

func Kind(t CatalogObjectType) string {
	switch t {
	case CatalogObjectTypeResource:
		return ResourceKind
	case CatalogObjectTypeSkillset:
		return SkillSetKind
	default:
		return ""
	}
}

func KindFromKindName(uri string) string {
	switch uri {
	case KindNameCatalogs:
		return CatalogKind
	case KindNameVariants:
		return VariantKind
	case KindNameNamespaces:
		return NamespaceKind
	case KindNameViews:
		return ViewKind
	case KindNameResources:
		return ResourceKind
	case KindNameSkillsets:
		return SkillSetKind
	default:
		return InvalidKind
	}
}

func KindNameFromObjectType(t CatalogObjectType) string {
	switch t {
	case CatalogObjectTypeResource:
		return "resources"
	case CatalogObjectTypeSkillset:
		return "skillsets"
	default:
		return ""
	}
}

func IsCatalogLevelKind(kind string) bool {
	return kind == KindNameViews
}

type CatalogObjectType string

const (
	CatalogObjectTypeInvalid  CatalogObjectType = "invalid"
	CatalogObjectTypeUnknown  CatalogObjectType = "unknown"
	CatalogObjectTypeResource CatalogObjectType = "resource"
	CatalogObjectTypeSkillset CatalogObjectType = "skillset"
)

func CatalogObjectTypeFromKind(k string) CatalogObjectType {
	switch k {
	case ResourceKind:
		return CatalogObjectTypeResource
	case SkillSetKind:
		return CatalogObjectTypeSkillset
	default:
		return CatalogObjectTypeInvalid
	}
}

const (
	ResourcePropertyDefinition = "definition"
	ResourcePropertyValue      = "value"
)

const (
	StdioRunnerID = "system.stdiorunner"
)

type TokenType string

const (
	IdentityTokenType TokenType = "id"
	AccessTokenType   TokenType = "access"
	UnknownTokenType  TokenType = "unknown"
)

type TokenVersion string

const (
	TokenVersionV0_1 TokenVersion = "0.1"
)
