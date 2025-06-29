package auth

import (
	"encoding/base64"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/auth/keymanager"
	"github.com/tansive/tansive/internal/common/httpx"
)

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
}

// GetJWKSHandler returns a handler that serves the JWKS endpoint
func GetJWKSHandler(km keymanager.KeyManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Ctx(r.Context()).Debug().Msg("Serving JWKS request")

		key, err := km.GetActiveKey(r.Context())
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("No active key available")
			httpx.SendJsonRsp(r.Context(), w, http.StatusInternalServerError, map[string]string{
				"error": "no active key available",
			})
			return
		}

		jwk := JWK{
			Kty: "OKP", // Octet Key Pair
			Kid: key.KeyID.String(),
			Use: "sig",
			Alg: "EdDSA",
			Crv: "Ed25519",
			X:   base64.RawURLEncoding.EncodeToString(key.PublicKey),
		}

		jwks := JWKS{
			Keys: []JWK{jwk},
		}

		httpx.SendJsonRsp(r.Context(), w, http.StatusOK, jwks)
	}
}
