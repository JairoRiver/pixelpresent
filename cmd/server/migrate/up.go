package migrate

import (
	"database/sql"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/migrations"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newMigrateUpCommand() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := util.LoadConfig(configFile)
			if err != nil {
				log.Fatal().Err(err).Msg("cannot load config")
			}
			util.SetupLogger(config)

			db, err := sql.Open("pgx", config.Database.DSN)
			if err != nil {
				log.Fatal().Err(err).Msg("cannot connect to database")
			}
			defer db.Close()

			d, err := iofs.New(migrations.MigrationsFS, ".")
			if err != nil {
				log.Fatal().Err(err).Msg("cannot load migration files")
			}

			driver, err := postgres.WithInstance(db, &postgres.Config{})
			if err != nil {
				log.Fatal().Err(err).Msg("cannot create postgres migrate instance")
			}

			m, err := migrate.NewWithInstance("iofs", d, "pixelpresent", driver)
			if err != nil {
				log.Fatal().Err(err).Msg("cannot create migrate instance")
			}

			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				log.Fatal().Err(err).Msg("migrate up failed")
			}

			log.Info().Msg("database migrated up successfully")
		},
	}

	cmd.Flags().StringVar(&configFile, "config", util.DefaultConfigPath, "config file")

	return cmd
}
