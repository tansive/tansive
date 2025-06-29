package config

const (
	// Support schemas arranged in hierarchical structure like values. While this retains existing code, this is not expected
	// to be enabled. Therefore, leave it false always.
	HierarchicalSchemas = false
	// Enable snappy compression for catalog objects to save space in the database.
	CompressCatalogObjects = true
	// Remap attribute schema references when a new attribute schema is created in a namespace when there are existing collections schemas
	// that refer to the same schema name in the root namespace. We'll set it to false by default to avoid unexpected behavior.
	RemapAttributeSchemaReferences = false
)
