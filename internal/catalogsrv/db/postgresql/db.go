// Description: This file contains the implementation of the hatchCatalogDb interface for the PostgreSQL database.
package postgresql

import (
	"github.com/tansive/tansive/internal/catalogsrv/db/dbmanager"
)

type hatchCatalogDb struct {
	mm *metadataManager
	om *objectManager
	cm *connectionManager
}

func NewHatchCatalogDb(c dbmanager.ScopedConn) (*metadataManager, *objectManager, *connectionManager) {
	h := &hatchCatalogDb{}
	h.mm = newMetadataManager(c)
	h.om = newObjectManager(c)
	h.cm = newConnectionManager(c)
	h.om.m = h.mm
	return h.mm, h.om, h.cm
}
