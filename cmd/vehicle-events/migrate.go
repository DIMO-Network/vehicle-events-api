package main

import (
	"context"
	"github.com/DIMO-Network/vehicle-events-api/internal/config"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"

	"github.com/DIMO-Network/shared/db"
	_ "github.com/lib/pq"
)

func migrateDatabase(ctx context.Context, logger zerolog.Logger, s *config.Settings, args []string) {
	command := "up"
	if len(args) > 2 {
		command = args[2]
		if command == "down-to" || command == "up-to" {
			command = command + " " + args[3]
		}
	}

	sqlDb := db.NewDbConnectionFromSettings(ctx, &s.DB, true)
	sqlDb.WaitForDB(logger)

	if command == "" {
		command = "up"
	}

	_, err := sqlDb.DBS().Writer.Exec("CREATE SCHEMA IF NOT EXISTS vehicle_events_api;")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create schema:")
	}
	goose.SetTableName("vehicle_events_api.migrations")
	if err := goose.RunContext(ctx, command, sqlDb.DBS().Writer.DB, "internal/infrastructure/db/migrations"); err != nil {
		logger.Fatal().Err(err).Msg("failed to apply go code migrations")
	}
}
