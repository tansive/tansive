package hashlog

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

func VerifyHashedLog(r io.Reader, pubKey []byte) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid ed25519 public key size: got %d", len(pubKey))
	}

	scanner := bufio.NewScanner(r)
	// Increase buffer size to handle larger lines (default is 64KB)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNum := 0
	expectedPrevHash := ""

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		var entry HashedLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}

		// Verify hash
		hashInput := struct {
			Payload  map[string]any `json:"payload"`
			PrevHash string         `json:"prevHash"`
		}{
			Payload:  entry.Payload,
			PrevHash: entry.PrevHash,
		}
		hashData, err := json.Marshal(hashInput)
		if err != nil {
			return fmt.Errorf("line %d: failed to marshal hash input: %w", lineNum, err)
		}
		computedHash := fmt.Sprintf("%x", sha256.Sum256(hashData))
		if entry.Hash != computedHash {
			return fmt.Errorf("line %d: hash mismatch", lineNum)
		}

		// Verify hash chain
		if entry.PrevHash != expectedPrevHash {
			return fmt.Errorf("line %d: prevHash mismatch", lineNum)
		}

		// Verify signature
		signInput := struct {
			Payload  map[string]any `json:"payload"`
			PrevHash string         `json:"prevHash"`
			Hash     string         `json:"hash"`
		}{
			Payload:  entry.Payload,
			PrevHash: entry.PrevHash,
			Hash:     entry.Hash,
		}
		signData, err := json.Marshal(signInput)
		if err != nil {
			return fmt.Errorf("line %d: failed to marshal signature input: %w", lineNum, err)
		}
		signature, err := base64.StdEncoding.DecodeString(entry.Signature)
		if err != nil {
			return fmt.Errorf("line %d: invalid base64 signature: %w", lineNum, err)
		}
		if !ed25519.Verify(pubKey, signData, signature) {
			return fmt.Errorf("line %d: signature verification failed", lineNum)
		}

		expectedPrevHash = entry.Hash
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read stream: %w", err)
	}

	return nil
}
