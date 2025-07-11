package db

import (
	"context"
	sharedDB "github.com/DIMO-Network/shared/pkg/db"
	"github.com/DIMO-Network/vehicle-events-api/internal/config"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

func MigrateDatabase(ctx context.Context, logger zerolog.Logger, s *config.Settings, args []string) {
	command := "up"
	if len(args) > 2 {
		command = args[2]
		if command == "down-to" || command == "up-to" {
			command = command + " " + args[3]
		}
	}

	sqlDb := sharedDB.NewDbConnectionFromSettings(ctx, &s.DB, true)
	sqlDb.WaitForDB(logger)

	if command == "" {
		command = "up"
	}

	_, err := sqlDb.DBS().Writer.Exec("CREATE SCHEMA IF NOT EXISTS vehicle_events_api;")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create schema:")
	}
	goose.SetTableName("vehicle_events_api.migrations")
	if err := goose.RunContext(ctx, command, sqlDb.DBS().Writer.DB, "internal/db/migrations"); err != nil {
		logger.Fatal().Err(err).Msg("failed to apply migrations")
	}
}
