package config

import (
	"github.com/tansive/tansive/internal/catalogsrv/config"
)

// HatchCatalogDsn returns the DSN for the Hatch Catalog database
func HatchCatalogDsn() string {
	return config.HatchCatalogDSN()
}

const CompressCatalogObjects = config.CompressCatalogObjects
