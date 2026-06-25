package migrate

import "github.com/spf13/cobra"

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Database migration helpers",
	}
}

func RegisterCommands(parent *cobra.Command) {
	cmd := newMigrateCmd()
	parent.AddCommand(cmd)
	cmd.AddCommand(newMigrateUpCommand())
	cmd.AddCommand(newMigrateDownCommand())
}
