package cmd

import (
	"github.com/nhurel/dim/lib"
	"github.com/spf13/cobra"
)

var labelRemoveCommand = &cobra.Command{
	Use:   "remove IMAGE[:TAG] LABEL_KEY...",
	Short: "Remove a label from a given image",
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		labels := args[1:]

		var imageTags []string
		var tag string
		var err error

		if PullFlag {
			if err = Dim.Pull(image); err != nil {
				return err
			}
		}

		if _, imageTags, err = Dim.GetImageInfo(image); err != nil {
			return err
		}

		if tag, err = guessTag(ImageFlag, image, imageTags, OverrideFlag); err != nil {
			return err
		}

		if err = Dim.RemoveLabel(image, labels, tag); err != nil {
			return err
		}

		if OverrideFlag || DeleteFlag {
			if dim.ListContains(imageTags, image) && image != tag {
				Dim.Remove(image)
			}
		}

		return nil
	},
}

func init() {
	labelRemoveCommand.Flags().StringVarP(&ImageFlag, "tag", "t", "", "Tag the new labeled image")
	labelRemoveCommand.Flags().BoolVarP(&DeleteFlag, "delete", "d", false, "Delete the original image both locally and on the registry")
	labelRemoveCommand.Flags().BoolVarP(&OverrideFlag, "override", "o", false, "Delete the original image locally only")
	labelRemoveCommand.Flags().BoolVarP(&PullFlag, "pull", "p", false, "Pull the image before adding label to ensure label is added to latest version")

	labelCommand.AddCommand(labelRemoveCommand)
}
