package catalogmanager

import (
	"context"

	"encoding/json"

	"github.com/tansive/tansive/internal/catalogsrv/catalogmanager/interfaces"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/pkg/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var _ = canonicalizeMetadata
var _ = getMetadata

func getMetadata(ctx context.Context, resourceJSON []byte) (*interfaces.Metadata, apperrors.Error) {
	if len(resourceJSON) == 0 {
		return nil, ErrEmptySchema
	}

	if !gjson.ValidBytes(resourceJSON) {
		return nil, ErrSchemaValidation
	}

	metadata := gjson.GetBytes(resourceJSON, "metadata")
	if !metadata.Exists() {
		return nil, ErrSchemaValidation
	}

	var schemaMetadata interfaces.Metadata
	if err := json.Unmarshal([]byte(metadata.Raw), &schemaMetadata); err != nil {
		return nil, ErrSchemaValidation
	}

	return &schemaMetadata, nil
}

func canonicalizeMetadata(resourceJSON []byte, kind string, metadata *interfaces.Metadata) ([]byte, *interfaces.Metadata, apperrors.Error) {
	if len(resourceJSON) == 0 {
		return nil, nil, ErrEmptySchema
	}

	if !gjson.ValidBytes(resourceJSON) {
		return nil, nil, ErrSchemaValidation.Msg("invalid JSON")
	}

	metadataResult := gjson.GetBytes(resourceJSON, "metadata")
	if !metadataResult.Exists() {
		return nil, nil, ErrSchemaValidation.Msg("missing metadata in resource schema")
	}

	var resourceMetadata interfaces.Metadata
	if err := json.Unmarshal([]byte(metadataResult.Raw), &resourceMetadata); err != nil {
		return nil, nil, ErrSchemaValidation.Msg("failed to unmarshal metadata")
	}

	if metadata != nil {
		if metadata.Name != "" {
			resourceMetadata.Name = metadata.Name
		}
		if metadata.Catalog != "" {
			resourceMetadata.Catalog = metadata.Catalog
		}
		if !metadata.Variant.IsNil() {
			resourceMetadata.Variant = metadata.Variant
		}
		if metadata.Path != "" {
			resourceMetadata.Path = metadata.Path
		}
		if metadata.Description != "" {
			resourceMetadata.Description = metadata.Description
		}
		if !metadata.Namespace.IsNil() {
			resourceMetadata.Namespace = metadata.Namespace
		}
	}

	if resourceMetadata.Variant.IsNil() {
		resourceMetadata.Variant = types.NullableStringFrom(catcommon.DefaultVariant) // set default variant if nil
	}

	updatedJSON, err := sjson.SetBytes(resourceJSON, "metadata", resourceMetadata)
	if err != nil {
		return nil, nil, ErrSchemaValidation.Msg("failed to update metadata")
	}

	return updatedJSON, &resourceMetadata, nil
}
