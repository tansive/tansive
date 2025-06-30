package catalogmanager

import (
	"context"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/catalogsrv/schema/schemavalidator"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tidwall/gjson"
)

func RequestType(resourceJSON []byte) (kind string, err apperrors.Error) {
	if !gjson.ValidBytes(resourceJSON) {
		return "", ErrInvalidSchema.Msg("invalid JSON format")
	}

	result := gjson.GetBytes(resourceJSON, "kind")
	if !result.Exists() {
		return "", ErrInvalidSchema.Msg("missing kind field")
	}
	kind = result.String()

	result = gjson.GetBytes(resourceJSON, "version")
	if !result.Exists() {
		return "", ErrInvalidSchema.Msg("missing version field")
	}
	version := result.String()

	if schemavalidator.ValidateKind(kind) && version == catcommon.ApiVersion {
		return kind, nil
	}
	return "", ErrInvalidSchema.Msg("invalid kind or version")
}

var kindHandlerFactories = map[string]interfaces.KindHandlerFactory{
	catcommon.CatalogKind:   NewCatalogKindHandler,
	catcommon.VariantKind:   NewVariantKindHandler,
	catcommon.NamespaceKind: NewNamespaceKindHandler,
	catcommon.ResourceKind:  NewResourceKindHandler,
	catcommon.SkillSetKind:  NewSkillSetKindHandler,
	catcommon.ViewKind:      policy.NewViewKindHandler,
}

func ResourceManagerForKind(ctx context.Context, kind string, name interfaces.RequestContext) (interfaces.KindHandler, apperrors.Error) {
	factory, ok := kindHandlerFactories[kind]
	if !ok {
		return nil, ErrInvalidSchema.Msg("unsupported resource kind: " + kind)
	}
	return factory(ctx, name)
}
