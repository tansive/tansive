package tangent

import (
	"context"
	"encoding/json"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

type TangentInfo struct {
	ID                     uuid.UUID            `json:"id"`
	CreatedBy              string               `json:"createdBy"`
	URL                    string               `json:"url"`
	Capabilities           []catcommon.RunnerID `json:"capabilities"`
	PublicKeyAccessKey     []byte               `json:"publicKeyAccessKey"`
	PublicKeyLogSigningKey []byte               `json:"publicKeyLogSigningKey"`
}

type Tangent struct {
	ID uuid.UUID `json:"id"`
	TangentInfo
}

func GetTangentWithCapabilities(ctx context.Context, capabilities []catcommon.RunnerID) (*Tangent, apperrors.Error) {
	if config.IsTest() {
		return &Tangent{
			ID: uuid.New(),
			TangentInfo: TangentInfo{
				CreatedBy:    "system",
				URL:          "http://local.tansive.dev:8468",
				Capabilities: capabilities,
			},
		}, nil
	}

	// Get all tangents
	tangents, err := db.DB(ctx).ListTangents(ctx)
	if err != nil {
		return nil, err
	}

	info := TangentInfo{}
	goerr := json.Unmarshal(tangents[0].Info, &info)
	if goerr != nil {
		return nil, apperrors.New("failed to unmarshal tangent info: " + goerr.Error())
	}

	return &Tangent{
		ID: info.ID,
		TangentInfo: TangentInfo{
			CreatedBy:    "system",
			URL:          info.URL,
			Capabilities: capabilities,
		},
	}, nil
}
