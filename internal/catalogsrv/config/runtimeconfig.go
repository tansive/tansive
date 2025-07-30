package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

type RuntimeConfig struct {
	EncryptionKey    string `json:"encryption_key"`
	UserPasswordHash string `json:"user_password_hash"`
}

func generateRandomPassword(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive, got %d", length)
	}

	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	password := base64.URLEncoding.EncodeToString(randomBytes)

	if len(password) > length {
		password = password[:length]
	}

	return password, nil
}

func getRuntimeConfigDir() string {
	return Config().RuntimeConfigDir + "/.tansivesrv"
}

func SetSingleUserPassword(password string) error {
	runtimeConfigDir := getRuntimeConfigDir()
	runtimeConfigFile := filepath.Join(runtimeConfigDir, "runtime_config.json")

	if _, err := os.Stat(runtimeConfigFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("runtime config file does not exist: %s", runtimeConfigFile)
		}
		return fmt.Errorf("error checking runtime config file: %w", err)
	}

	file, err := os.Open(runtimeConfigFile)
	if err != nil {
		return fmt.Errorf("error opening runtime config file: %w", err)
	}
	defer file.Close()

	var runtimeConfig RuntimeConfig
	if err := json.NewDecoder(file).Decode(&runtimeConfig); err != nil {
		return fmt.Errorf("error decoding runtime config file: %w", err)
	}

	runtimeConfig.UserPasswordHash = password

	file, err = os.OpenFile(runtimeConfigFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error creating/opening runtime config file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(runtimeConfig); err != nil {
		return fmt.Errorf("error encoding runtime config: %w", err)
	}

	Config().SingleUserPasswordHash = password

	log.Info().Msg("User password updated in runtime config")
	return nil
}

func Init() {
	log.Info().Msg("Initializing runtime config")
	auditLogPath := Config().AuditLog.GetPath()
	log.Info().Msgf("Audit log path: %s", auditLogPath)
	err := os.MkdirAll(auditLogPath, 0755)
	if err != nil {
		log.Error().Err(err).Msg("Error creating audit log path")
		os.Exit(1)
	}

	runtimeConfigDir := getRuntimeConfigDir()
	log.Info().Msgf("Runtime config dir: %s", runtimeConfigDir)
	err = os.MkdirAll(runtimeConfigDir, 0755)
	if err != nil {
		log.Error().Err(err).Msg("Error creating runtime config dir")
		os.Exit(1)
	}

	runtimeConfigFile := filepath.Join(runtimeConfigDir, "runtime_config.json")

	if _, err := os.Stat(runtimeConfigFile); err == nil {
		log.Info().Msg("Runtime config file exists, reading values")
		file, err := os.Open(runtimeConfigFile)
		if err != nil {
			log.Error().Err(err).Msg("Error opening runtime config file")
			os.Exit(1)
		}
		defer file.Close()

		var runtimeConfig RuntimeConfig
		if err := json.NewDecoder(file).Decode(&runtimeConfig); err != nil {
			log.Error().Err(err).Msg("Error decoding runtime config file")
			os.Exit(1)
		}

		if runtimeConfig.EncryptionKey != "" {
			Config().Auth.KeyEncryptionPasswd = runtimeConfig.EncryptionKey
			log.Info().Msg("Set KeyEncryptionPasswd from runtime config")
		}
		if runtimeConfig.UserPasswordHash != "" {
			Config().SingleUserPasswordHash = runtimeConfig.UserPasswordHash
			log.Info().Msg("Set SingleUserPassword from runtime config")
		}
	} else if os.IsNotExist(err) {
		log.Info().Msg("Runtime config file doesn't exist, creating with new encryption key")

		encryptionKey, err := generateRandomPassword(64)
		if err != nil {
			log.Error().Err(err).Msg("Error generating random encryption key")
			os.Exit(1)
		}

		Config().Auth.KeyEncryptionPasswd = encryptionKey

		runtimeConfig := RuntimeConfig{
			EncryptionKey: encryptionKey,
		}

		file, err := os.OpenFile(runtimeConfigFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			log.Error().Err(err).Msg("Error creating runtime config file")
			os.Exit(1)
		}
		defer file.Close()

		if err := json.NewEncoder(file).Encode(runtimeConfig); err != nil {
			log.Error().Err(err).Msg("Error encoding runtime config")
			os.Exit(1)
		}

		log.Info().Msgf("Created runtime config file with new encryption key: %s", runtimeConfigFile)
	} else {
		log.Error().Err(err).Msg("Error checking runtime config file")
		os.Exit(1)
	}
}
