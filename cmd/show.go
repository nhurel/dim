package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var showCommand = &cobra.Command{
	Use: "show image",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("image name is missing")
		}
		image := args[0]
		return Dim.PrintImageInfo(os.Stdout, image, templateFlag)
	},
}

var templateFlag string

func init() {
	//TODO Add --output flag to write in a file
	showCommand.Flags().StringVarP(&templateFlag, "template", "t", "", "Template to use to display image info")
	RootCommand.AddCommand(showCommand)
}
