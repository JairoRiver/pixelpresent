package serve

import (
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the pixelpresent server",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := util.LoadConfig(configFile)
			if err != nil {
				log.Fatal().Err(err).Msg("cannot load config")
			}
			util.SetupLogger(config)

			log.Info().Msg("pixelpresent server")
		},
	}

	cmd.Flags().StringVar(&configFile, "config", util.DefaultConfigPath, "config file")

	return cmd
}

func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(newServeCommand())
}
