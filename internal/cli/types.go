package cli

const (
	KindCatalog   = "Catalog"
	KindVariant   = "Variant"
	KindNamespace = "Namespace"
	KindView      = "View"
	KindSkillset  = "SkillSet"
	KindResource  = "Resource"
)

func ValidateResourceKind(kind string) bool {
	switch kind {
	case KindCatalog, KindVariant, KindNamespace, KindView, KindSkillset, KindResource:
		return true
	default:
		return false
	}
}
