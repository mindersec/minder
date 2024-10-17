// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package app provides the entrypoint for the minder migrations
package app

import (
	"context"
	"errors"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/config"
	serverconfig "github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/logger"
)

// number of secrets to re-encrypt per batch
const batchSize = 100

// used if rotation is cancelled before commit for one reason or another
var errCancelRotation = errors.New("cancelling rotation process")

// rotateCmd represents the `encryption rotate` command
var rotateCmd = &cobra.Command{
	Use:   "rotate-provider-tokens",
	Short: "Rotate keys and encryption algorithms for provider tokens",
	Long:  `re-encrypt all provider access tokens with the default key version and algorithm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
		if err != nil {
			cliErrorf(cmd, "unable to read config: %s", err)
		}

		// ensure that the new config structure is set - otherwise bad things will happen
		if cfg.Crypto.Default.KeyID == "" {
			cliErrorf(cmd, "default key ID not defined in crypto config - exiting")
		}

		ctx := logger.FromFlags(cfg.LoggingConfig).WithContext(context.Background())

		zerolog.Ctx(ctx).Debug().
			Str("default_key_id", cfg.Crypto.Default.KeyID).
			Str("default_algorithm", string(crypto.DefaultAlgorithm)).
			Msg("default encryption settings")

		// instantiate `db.Store` so we can run queries
		store, closer, err := wireUpDB(ctx, cfg)
		if err != nil {
			cliErrorf(cmd, "unable to connect to database: %s", err)
		}
		defer closer()

		yes := confirm(cmd, "Running this command will re-encrypt provider access tokens")
		if !yes {
			return nil
		}

		// rotate the provider access tokens
		totalRotated, err := rotateSecrets(ctx, cmd, store, cfg)
		if err != nil {
			// if we cancel or have nothing to migrate...
			if errors.Is(err, errCancelRotation) {
				cmd.Printf("Nothing to migrate, exiting\n")
				return nil
			}
			cliErrorf(cmd, "error while attempting to rotate secrets: %s", err)
		}

		cmd.Printf("Successfully rotated %d secrets\n", totalRotated)
		return nil
	},
}

func rotateSecrets(
	ctx context.Context,
	cmd *cobra.Command,
	store db.Store,
	cfg *serverconfig.Config,
) (int64, error) {
	// instantiate crypto engine so that we can decrypt and re-encrypt
	cryptoEngine, err := crypto.NewEngineFromConfig(cfg)
	if err != nil {
		cliErrorf(cmd, "unable to instantiate crypto engine: %s", err)
	}

	return db.WithTransaction[int64](store, func(qtx db.ExtendQuerier) (int64, error) {
		var rotated int64 = 0

		for {
			updated, err := runRotationBatch(ctx, rotated, qtx, cryptoEngine, &cfg.Crypto)
			if err != nil {
				return rotated, err
			}
			// nothing more to do - exit loop
			if updated == 0 {
				break
			}
			rotated += updated
		}

		// print useful status for the case where there is nothing to rotate
		if rotated == 0 {
			return 0, errCancelRotation
		}
		// one last chance to reconsider your choices
		yes := confirm(cmd, fmt.Sprintf("About to rotate %d secrets, do you want to continue?", rotated))
		if !yes {
			return 0, errCancelRotation
		}
		return rotated, nil
	})
}

func runRotationBatch(
	ctx context.Context,
	offset int64,
	store db.ExtendQuerier,
	engine crypto.Engine,
	cfg *serverconfig.CryptoConfig,
) (int64, error) {
	batch, err := store.ListTokensToMigrate(ctx, db.ListTokensToMigrateParams{
		DefaultAlgorithm:  string(crypto.DefaultAlgorithm),
		DefaultKeyVersion: cfg.Default.KeyID,
		BatchOffset:       offset,
		BatchSize:         batchSize,
	})
	if err != nil {
		return 0, err
	}

	zerolog.Ctx(ctx).
		Debug().
		Msgf("processing batch of %d tokens", len(batch))

	for _, token := range batch {
		var oldSecret crypto.EncryptedData
		if token.EncryptedAccessToken.Valid {
			deserialized, err := crypto.DeserializeEncryptedData(token.EncryptedAccessToken.RawMessage)
			if err != nil {
				return 0, tokenError(token.ID, "secret deserialization", err)
			}
			oldSecret = deserialized
		} else if token.EncryptedToken.Valid {
			oldSecret = crypto.NewBackwardsCompatibleEncryptedData(token.EncryptedToken.String)
		} else {
			// this should never happen
			return 0, tokenError(token.ID, "secret retrieval", errors.New("no encrypted secret found"))
		}

		zerolog.Ctx(ctx).Debug().
			Int32("token_id", token.ID).
			Str("key_version", oldSecret.KeyVersion).
			Str("algorithm", string(oldSecret.Algorithm)).
			Msg("re-encrypting old secret")

		// decrypt the secret
		decrypted, err := engine.DecryptOAuthToken(oldSecret)
		if err != nil {
			return 0, tokenError(token.ID, "decryption", err)
		}

		// re-encrypt it with new key/algorithm
		encrypted, err := engine.EncryptOAuthToken(&decrypted)
		if err != nil {
			return 0, tokenError(token.ID, "encryption", err)
		}

		// update DB
		serialized, err := encrypted.Serialize()
		if err != nil {
			return 0, tokenError(token.ID, "secret serialization", err)
		}

		zerolog.Ctx(ctx).
			Debug().
			Msgf("updating provider token %d", token.ID)

		err = store.UpdateEncryptedSecret(ctx, db.UpdateEncryptedSecretParams{
			ID:     token.ID,
			Secret: serialized,
		})
		if err != nil {
			return 0, tokenError(token.ID, "secret update in database", err)
		}
	}

	return int64(len(batch)), nil
}

func tokenError(tokenID int32, action string, err error) error {
	return fmt.Errorf("unable to re-encrypt provider token %d during %s: %s", tokenID, action, err)
}

func init() {
	encryptionCmd.AddCommand(rotateCmd)
}
