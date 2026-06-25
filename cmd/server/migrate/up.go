package migrate

import (
	"database/sql"
	"log"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/migrations"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
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
				log.Fatalf("cannot load config: %v", err)
			}

			db, err := sql.Open("pgx", config.Database.DSN)
			if err != nil {
				log.Fatalf("cannot connect to database: %v", err)
			}
			defer db.Close()

			d, err := iofs.New(migrations.MigrationsFS, ".")
			if err != nil {
				log.Fatalf("cannot load migration files: %v", err)
			}

			driver, err := postgres.WithInstance(db, &postgres.Config{})
			if err != nil {
				log.Fatalf("cannot create postgres migrate instance: %v", err)
			}

			m, err := migrate.NewWithInstance("iofs", d, "pixelpresent", driver)
			if err != nil {
				log.Fatalf("cannot create migrate instance: %v", err)
			}

			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("migrate up failed: %v", err)
			}

			log.Println("database migrated up successfully")
		},
	}

	cmd.Flags().StringVar(&configFile, "config", util.DefaultConfigPath, "config file")

	return cmd
}
