package migrate

import (
	"database/sql"
	"strconv"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/migrations"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newMigrateDownCommand() *cobra.Command {
	var configFile string
	var steps int

	cmd := &cobra.Command{
		Use:   "down [steps]",
		Short: "Revert database migrations, given the number of steps",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			steps = n
			return nil
		},
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

			if steps == 0 {
				err = m.Down()
			} else {
				err = m.Steps(-1 * steps)
			}
			if err != nil && err != migrate.ErrNoChange {
				log.Fatal().Err(err).Msg("migrate down failed")
			}

			log.Info().Msg("database migrated down successfully")
		},
	}

	cmd.Flags().StringVar(&configFile, "config", util.DefaultConfigPath, "config file")

	return cmd
}
