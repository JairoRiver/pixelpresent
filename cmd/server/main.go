package main

import (
	"log"

	"github.com/JairoRiver/pixelpresent/cmd/server/migrate"
	"github.com/JairoRiver/pixelpresent/cmd/server/serve"
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
		log.Fatal(err)
	}
}
