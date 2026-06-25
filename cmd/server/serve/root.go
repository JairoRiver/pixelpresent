package serve

import (
	"log"

	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the pixelpresent server",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("pixelpresent server")
		},
	}
}

func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(newServeCommand())
}
