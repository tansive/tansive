package keymanager

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/config"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/dberror"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

// KeyManager defines the interface for key management operations
type KeyManager interface {
	GetActiveKey(ctx context.Context) (*SigningKey, apperrors.Error)
}

// The KeyManager implementation here is not intended for production use.
// This is only for testing purposes. It uses a password stored in the config file
// to encrypt a signing key which is stored in the database. Real production use
// should use a KMS

var (
	keyManagerInstance *keyManager
	keyManagerOnce     sync.Once
)

// GetKeyManager returns the singleton instance of KeyManager
func GetKeyManager() KeyManager {
	keyManagerOnce.Do(func() {
		keyManagerInstance = &keyManager{}
	})
	return keyManagerInstance
}

// SigningKey represents a key pair used for signing tokens
type SigningKey struct {
	KeyID      uuid.UUID
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	ExpiresAt  time.Time
}

// IsExpired checks if the signing key has expired
func (sk *SigningKey) IsExpired() bool {
	return sk.ExpiresAt.Before(time.Now())
}

// keyManager handles the management of signing keys
type keyManager struct {
	activeKey *SigningKey
	mu        sync.RWMutex
}

// GetActiveKey retrieves the active signing key, creating a new one if necessary
func (km *keyManager) GetActiveKey(ctx context.Context) (*SigningKey, apperrors.Error) {
	if km.activeKey != nil {
		return km.activeKey, nil
	}
	return km.retrieveOrCreateKey(ctx)
}

// retrieveOrCreateKey retrieves an existing key or creates a new one
func (km *keyManager) retrieveOrCreateKey(ctx context.Context) (*SigningKey, apperrors.Error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.activeKey != nil {
		return km.activeKey, nil
	}

	var key *models.SigningKey
	err := retry.Do(func() error {
		var err error
		key, err = db.DB(ctx).GetActiveSigningKey(ctx)
		if err != nil {
			if errors.Is(err, dberror.ErrNotFound) {
				return nil
			}
			return retry.Unrecoverable(err)
		}
		return err
	}, retry.Attempts(3), retry.Delay(1*time.Second), retry.DelayType(retry.BackOffDelay))
	if err != nil {
		return nil, apperrors.New("unable to retrieve signing key").Err(err)
	}

	if key == nil {
		// Create new key pair
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("unable to generate signing key")
			return nil, apperrors.New("unable to generate signing key").Err(err)
		}

		encKey, err := catcommon.Encrypt(priv, config.Config().Auth.KeyEncryptionPasswd)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("unable to encrypt signing key")
			return nil, apperrors.New("unable to encrypt signing key").Err(err)
		}

		key = &models.SigningKey{
			PublicKey:  pub,
			PrivateKey: encKey,
			IsActive:   true,
		}

		err = retry.Do(func() error {
			return db.DB(ctx).CreateSigningKey(ctx, key)
		}, retry.Attempts(3), retry.Delay(1*time.Second), retry.DelayType(retry.BackOffDelay))
		if err != nil {
			return nil, apperrors.New("unable to create signing key").Err(err)
		}

		km.activeKey = &SigningKey{
			KeyID:      key.KeyID,
			PrivateKey: priv,
			PublicKey:  pub,
		}
	} else {
		// Decrypt the existing key
		decKey, err := catcommon.Decrypt(key.PrivateKey, config.Config().Auth.KeyEncryptionPasswd)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("unable to decrypt signing key")
			return nil, apperrors.New("unable to decrypt signing key").Err(err)
		}

		km.activeKey = &SigningKey{
			KeyID:      key.KeyID,
			PrivateKey: decKey,
			PublicKey:  key.PublicKey,
		}
	}

	return km.activeKey, nil
}
