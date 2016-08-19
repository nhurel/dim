package cmd

import "github.com/spf13/cobra"

var genBashCompletionCommand = &cobra.Command{
	Hidden: true,
	Use:    "autocomplete",
	Run: func(cmd *cobra.Command, args []string) {
		RootCommand.GenBashCompletionFile("dim_compl")
	},
}

func init() {
	RootCommand.AddCommand(genBashCompletionCommand)
}
