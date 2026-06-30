package serve

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/JairoRiver/pixelpresent/internal/api"
	"github.com/JairoRiver/pixelpresent/internal/auth"
	"github.com/JairoRiver/pixelpresent/internal/email"
	"github.com/JairoRiver/pixelpresent/internal/gifts"
	"github.com/JairoRiver/pixelpresent/internal/reactions"
	"github.com/JairoRiver/pixelpresent/internal/repository"
	"github.com/JairoRiver/pixelpresent/internal/repository/db"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/JairoRiver/pixelpresent/internal/web"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// shutdownTimeout bounds how long in-flight requests have to finish on shutdown.
const shutdownTimeout = 10 * time.Second

func newServeCommand() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the pixelpresent server",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := util.LoadConfig(configFile)
			if err != nil {
				return err
			}
			util.SetupLogger(config)

			return run(config)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", util.DefaultConfigPath, "config file")

	return cmd
}

// run wires the dependencies and serves HTTP until an interrupt signal arrives.
func run(config util.Config) error {
	magicLinkTTL, err := time.ParseDuration(config.Auth.MagicLinkTTL)
	if err != nil {
		return errors.New("invalid auth.magic_link_ttl: " + err.Error())
	}
	sessionTTL, err := time.ParseDuration(config.Auth.SessionTTL)
	if err != nil {
		return errors.New("invalid auth.session_ttl: " + err.Error())
	}

	// Cancel the base context (and trigger shutdown) on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, config.Database.DSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	users := repository.NewUserRepo(pool)
	links := repository.NewMagicLinkRepo(pool)
	giftRepo := repository.NewGiftRepo(pool)
	reactionRepo := repository.NewReactionRepo(pool)
	emailSender := email.NewSMTPSender(email.SMTPConfig{
		Host:     config.Email.Host,
		Port:     config.Email.Port,
		Username: config.Email.Username,
		Password: config.Email.Password,
		From:     config.Email.From,
	})

	authService := auth.NewService(users, links, emailSender, config.App.BaseURL, magicLinkTTL)
	sessions := auth.NewSessionManager(config.Auth.SessionSecret, config.Environment == "production", sessionTTL)
	giftService := gifts.NewService(giftRepo)
	reactionService := reactions.NewService(giftRepo, reactionRepo)
	apiServer := api.NewServer(authService, sessions, giftService, reactionService)
	// API docs are a development affordance: never mounted in production.
	if config.Environment != "production" {
		apiServer.EnableDocs()
	}
	// Serve the embedded frontend. If the binary was built without building the
	// frontend, log a clear, actionable warning and keep serving the API only.
	if staticFS, err := web.Dist(); err != nil {
		log.Warn().Err(err).Msg("serving API without embedded frontend")
	} else {
		apiServer.ServeStatic(staticFS)
	}

	srv := &http.Server{
		Addr:              config.Server.Addr,
		Handler:           apiServer.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("pixelpresent server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Info().Msg("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(newServeCommand())
}
