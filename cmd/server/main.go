package main

import (
	"github.com/JairoRiver/pixelpresent/cmd/server/migrate"
	"github.com/JairoRiver/pixelpresent/cmd/server/serve"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "server",
	}

	serve.RegisterCommands(cmd)
	migrate.RegisterCommands(cmd)

	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		log.Fatal().Err(err).Msg("server exited with error")
	}
}
