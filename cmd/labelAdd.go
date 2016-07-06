package cmd

import (
	"github.com/nhurel/dim/lib"
	"github.com/spf13/cobra"
)

var labelAddCommand = &cobra.Command{
	Use:   "add IMAGE[:TAG] LABEL_KEY=LABEL_VALUE...",
	Short: "Add a label to a given image",
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

		if err = Dim.AddLabel(image, labels, tag); err != nil {
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

var (
	ImageFlag    string
	DeleteFlag   bool
	OverrideFlag bool
	PullFlag     bool
)

func init() {
	labelAddCommand.Flags().StringVarP(&ImageFlag, "tag", "t", "", "Tag the new labeled image")
	labelAddCommand.Flags().BoolVarP(&DeleteFlag, "delete", "d", false, "Delete the original image both locally and on the registry")
	labelAddCommand.Flags().BoolVarP(&OverrideFlag, "override", "o", false, "Delete the original image locally only")
	labelAddCommand.Flags().BoolVarP(&PullFlag, "pull", "p", false, "Pull the image before adding label to ensure label is added to latest version")

	labelCommand.AddCommand(labelAddCommand)
}
