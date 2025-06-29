package auth

var revocationChecker RevocationChecker = &defaultRevocationChecker{}

// RevocationChecker defines the interface for checking if a token has been revoked
type RevocationChecker interface {
	IsRevoked(jti string) bool
}

// defaultRevocationChecker is a simple implementation that always returns false
type defaultRevocationChecker struct{}

func (c *defaultRevocationChecker) IsRevoked(jti string) bool {
	return false
}

// SetRevocationChecker sets the revocation checker implementation
func SetRevocationChecker(checker RevocationChecker) {
	if checker == nil {
		checker = &defaultRevocationChecker{}
	}
	revocationChecker = checker
}
