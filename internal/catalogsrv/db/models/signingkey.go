package models

import (
	"time"

	"github.com/tansive/tansive/internal/common/uuid"
)

/*
   Column     |          Type           | Collation | Nullable |      Default
--------------+-------------------------+-----------+----------+--------------------
 key_id       | uuid                    |           | not null | uuid_generate_v4()
 public_key   | bytea                   |           | not null |
 private_key  | bytea                   |           | not null |
 is_active    | boolean                 |           | not null | false
 created_at   | timestamptz            |           | not null | now()
 updated_at   | timestamptz            |           | not null | now()
Indexes:
    "signing_keys_pkey" PRIMARY KEY, btree (key_id)
    "idx_active_signing_key" UNIQUE, btree (is_active) WHERE is_active = true
*/

type SigningKey struct {
	KeyID      uuid.UUID `db:"key_id"`
	PublicKey  []byte    `db:"public_key"`
	PrivateKey []byte    `db:"private_key"`
	IsActive   bool      `db:"is_active"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
