package cmd

import "github.com/spf13/cobra"

var labelCommand = &cobra.Command{
	Use: "label",
}

func init() {
	RootCommand.AddCommand(labelCommand)
}
