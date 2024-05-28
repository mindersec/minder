// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
)

// number of secrets to re-encrypt per batch
const batchSize = 100

// used if rotation is cancelled before commit for one reason or another
var errCancelRotation = errors.New("cancelling rotation process")

// rotateCmd represents the up command
var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate keys and encryption algorithms",
	Long:  `re-encrypt all provider access tokens with the default key version and algorithm`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
		if err != nil {
			cliErrorf(cmd, "unable to read config: %s", err)
		}

		// ensure that the new config structure is set - otherwise bad things will happen
		if cfg.Crypto.Default.KeyID == "" || cfg.Crypto.Default.Algorithm == "" {
			cliErrorf(cmd, "defaults not defined in crypto config - exiting")
		}

		ctx := logger.FromFlags(cfg.LoggingConfig).WithContext(context.Background())

		// instantiate `db.Store` so we can run queries
		store, closer, err := wireUpDB(ctx, cfg)
		if err != nil {
			cliErrorf(cmd, "unable to connect to database: %s", err)
		}
		defer closer()

		yes := confirm(cmd, "Running this command will change encrypted secrets")
		if !yes {
			return nil
		}

		// Clean up old session secrets instead of migrating
		sessionsDeleted, err := deleteStaleSessions(ctx, cmd, store)
		if err != nil {
			// if we cancel or have nothing to migrate...
			if errors.Is(err, errCancelRotation) {
				cmd.Printf("Cleanup canceled, exiting\n")
				return nil
			}
			cliErrorf(cmd, "error while deleting stale sessions: %s", err)
		}
		if sessionsDeleted != 0 {
			cmd.Printf("Successfully deleted %d stale sessions\n", sessionsDeleted)
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

func deleteStaleSessions(
	ctx context.Context,
	cmd *cobra.Command,
	store db.Store,
) (int64, error) {
	return db.WithTransaction[int64](store, func(qtx db.ExtendQuerier) (int64, error) {
		// delete any sessions more than one day old
		deleted, err := qtx.DeleteExpiredSessionStates(ctx)
		if err != nil {
			return 0, err
		}

		// skip the confirmation if there's nothing to do
		if deleted == 0 {
			cmd.Printf("No stale sessions to delete\n")
			return 0, nil
		}

		// one last chance to reconsider your choices
		yes := confirm(cmd, fmt.Sprintf("About to delete %d stale sessions", deleted))
		if !yes {
			return 0, errCancelRotation
		}
		return deleted, nil
	})
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
		DefaultAlgorithm:  cfg.Default.Algorithm,
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
				return 0, tokenError(token.ID, err)
			}
			oldSecret = deserialized
		} else if token.EncryptedToken.Valid {
			oldSecret = crypto.NewBackwardsCompatibleEncryptedData(token.EncryptedToken.String)
		} else {
			// this should never happen
			return 0, tokenError(token.ID, errors.New("no encrypted secret found"))
		}

		// decrypt the secret
		decrypted, err := engine.DecryptOAuthToken(oldSecret)
		if err != nil {
			return 0, tokenError(token.ID, err)
		}

		// re-encrypt it with new key/algorithm
		encrypted, err := engine.EncryptOAuthToken(&decrypted)
		if err != nil {
			return 0, tokenError(token.ID, err)
		}

		// update DB
		serialized, err := encrypted.Serialize()
		if err != nil {
			return 0, tokenError(token.ID, err)
		}

		zerolog.Ctx(ctx).
			Debug().
			Msgf("updating provider token %d", token.ID)

		err = store.UpdateEncryptedSecret(ctx, db.UpdateEncryptedSecretParams{
			ID:     token.ID,
			Secret: serialized,
		})
		if err != nil {
			return 0, tokenError(token.ID, err)
		}
	}

	return int64(len(batch)), nil
}

func tokenError(tokenID int32, err error) error {
	return fmt.Errorf("unable to re-encrypt provider token %d: %s", tokenID, err)
}

func init() {
	encryptionCmd.AddCommand(rotateCmd)
}
