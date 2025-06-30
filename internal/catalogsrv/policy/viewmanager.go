package policy

import (
	"context"
	"errors"

	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

type ViewManager interface {
	ID() uuid.UUID
	Name() string
	Scope() Scope
	GetViewDefinition() *ViewDefinition
	GetViewDefinitionJSON() ([]byte, apperrors.Error)
	GetResourcePath() (string, apperrors.Error)
	GetViewModel() (*models.View, apperrors.Error)
	CatalogID() uuid.UUID
}
type viewManager struct {
	view    *models.View
	viewDef *ViewDefinition
}

func NewViewManagerByViewLabel(ctx context.Context, viewLabel string) (ViewManager, apperrors.Error) {
	catalogID := catcommon.GetCatalogID(ctx)
	if catalogID == uuid.Nil || viewLabel == "" {
		return nil, ErrInvalidView
	}

	view, err := db.DB(ctx).GetViewByLabel(ctx, viewLabel, catalogID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrViewNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load view")
		return nil, ErrUnableToLoadObject.Msg("unable to load view")
	}
	viewDef, err := unmarshalViewDefinition(view)
	if err != nil {
		return nil, err
	}

	viewManager := &viewManager{view: view, viewDef: viewDef}
	return viewManager, nil
}

func NewViewManagerByViewID(ctx context.Context, viewID uuid.UUID) (ViewManager, apperrors.Error) {
	view, err := db.DB(ctx).GetView(ctx, viewID)
	if err != nil {
		if errors.Is(err, dberror.ErrNotFound) {
			return nil, ErrViewNotFound
		}
		log.Ctx(ctx).Error().Err(err).Msg("failed to load view")
		return nil, ErrUnableToLoadObject.Msg("unable to load view")
	}
	viewDef, err := unmarshalViewDefinition(view)
	if err != nil {
		return nil, err
	}
	return &viewManager{view: view, viewDef: viewDef}, nil
}

func (v *viewManager) ID() uuid.UUID {
	return v.view.ViewID
}

func (v *viewManager) Name() string {
	return v.view.Label
}

func (v *viewManager) GetViewDefinition() *ViewDefinition {
	return v.viewDef
}

func (v *viewManager) GetViewModel() (*models.View, apperrors.Error) {
	return v.view, nil
}

func (v *viewManager) CatalogID() uuid.UUID {
	return v.view.CatalogID
}

func (v *viewManager) GetViewDefinitionJSON() ([]byte, apperrors.Error) {
	if v.viewDef == nil {
		return nil, ErrInvalidView.Msg("view definition is nil")
	}
	json, err := v.viewDef.ToJSON()
	if err != nil {
		return nil, ErrInvalidView.Msg("unable to marshal view definition")
	}
	return json, nil
}

func (v *viewManager) Scope() Scope {
	if v.viewDef == nil {
		return Scope{}
	}
	return v.viewDef.Scope
}

func (v *viewManager) GetResourcePath() (string, apperrors.Error) {
	return "/views/" + v.view.Label, nil
}

func unmarshalViewDefinition(view *models.View) (*ViewDefinition, apperrors.Error) {
	if view == nil {
		return nil, ErrInvalidView.Msg("view is nil")
	}
	var viewDef ViewDefinition
	if err := json.Unmarshal(view.Rules, &viewDef); err != nil {
		return nil, ErrUnableToLoadObject.Msg("unable to unmarshal view rules")
	}
	return &viewDef, nil
}
