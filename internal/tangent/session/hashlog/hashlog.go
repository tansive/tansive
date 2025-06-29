// Package hashlog provides cryptographically signed log writing functionality.
// It creates tamper-evident logs with hash chains and digital signatures for audit trails.
// The package requires valid Ed25519 private keys for signing and supports configurable flush intervals.
package hashlog

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	jsonitor "github.com/json-iterator/go"
)

var json = jsonitor.ConfigCompatibleWithStandardLibrary

// HashedLogEntry represents a single signed log entry in the hash chain.
// Contains the payload, previous hash, current hash, and digital signature.
type HashedLogEntry struct {
	Payload   map[string]any `json:"payload"`   // log entry data
	PrevHash  string         `json:"prevHash"`  // hash of previous entry
	Hash      string         `json:"hash"`      // hash of current entry
	Signature string         `json:"signature"` // digital signature of entry
}

// HashLogWriter provides functionality to write cryptographically signed log entries.
// Maintains a hash chain and signs entries with Ed25519 for tamper detection.
type HashLogWriter struct {
	file          *os.File
	path          string
	flushInterval int
	mu            sync.Mutex
	buffer        []HashedLogEntry
	prevHash      string
	privKey       []byte
	closed        bool
}

// NewHashLogWriter creates a new hash log writer with the specified configuration.
// Returns the writer instance and any error encountered during creation.
// Private key must be exactly 32 bytes for Ed25519 signing.
func NewHashLogWriter(path string, flushInterval int, privKey []byte) (*HashLogWriter, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key: must be %d bytes, got %d", ed25519.PrivateKeySize, len(privKey))
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &HashLogWriter{
		file:          f,
		path:          path,
		flushInterval: flushInterval,
		buffer:        make([]HashedLogEntry, 0, flushInterval),
		privKey:       privKey,
	}, nil
}

// AddEntry adds a new signed log entry to the hash chain.
// Computes hash, signs the entry, and buffers it for writing.
// Returns an error if signing or hashing fails.
func (lw *HashLogWriter) AddEntry(payload map[string]any) error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	cloned := make(map[string]any, len(payload))
	for k, v := range payload {
		cloned[k] = v
	}

	entry := HashedLogEntry{
		Payload:  cloned,
		PrevHash: lw.prevHash,
	}

	// Compute hash
	dataToHash, err := json.Marshal(struct {
		Payload  map[string]any `json:"payload"`
		PrevHash string         `json:"prevHash"`
	}{
		Payload:  entry.Payload,
		PrevHash: entry.PrevHash,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}
	hash := sha256.Sum256(dataToHash)
	entry.Hash = fmt.Sprintf("%x", hash[:])
	lw.prevHash = entry.Hash

	// Sign (Payload + PrevHash + Hash)
	signInput, err := json.Marshal(struct {
		Payload  map[string]any `json:"payload"`
		PrevHash string         `json:"prevHash"`
		Hash     string         `json:"hash"`
	}{
		Payload:  entry.Payload,
		PrevHash: entry.PrevHash,
		Hash:     entry.Hash,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal sign input: %w", err)
	}
	signature := ed25519.Sign(lw.privKey, signInput)
	entry.Signature = base64.StdEncoding.EncodeToString(signature)

	lw.buffer = append(lw.buffer, entry)
	if len(lw.buffer) >= lw.flushInterval {
		return lw.flushLocked()
	}
	return nil
}

// flushLocked writes buffered entries to the log file.
// Must be called with the mutex locked.
func (lw *HashLogWriter) flushLocked() error {
	for _, entry := range lw.buffer {
		b, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		if _, err := lw.file.Write(append(b, '\n')); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}
	lw.buffer = lw.buffer[:0]
	return nil
}

// Flush writes all buffered entries to the log file.
// Returns an error if writing fails.
func (lw *HashLogWriter) Flush() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.flushLocked()
}

// Close flushes remaining entries and closes the log file.
// Ensures all buffered data is written before closing.
func (lw *HashLogWriter) Close() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.closed {
		return nil
	}

	if err := lw.flushLocked(); err != nil {
		return err
	}

	err := lw.file.Close()
	lw.closed = true
	return err
}
