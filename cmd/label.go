package cmd

import (
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib"
	"github.com/spf13/cobra"
)

var labelCommand = &cobra.Command{
	Use:   "label [--delete] IMAGE[:TAG] LABEL_KEY=LABEL_VALUE...",
	Short: "Add / Remove a label to a given image",
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		labels := args[1:]

		var imageTags []string
		var tag string
		var err error

		var authConfig *types.AuthConfig
		if RemoteFlag {
			if username != "" || password != "" {
				authConfig = &types.AuthConfig{Username: username, Password: password}
			}
			// TODO : get credentials the docker way and/or handle login
		}

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

		if DeleteFlag {
			if err = Dim.RemoveLabel(image, labels, tag); err != nil {
				return err
			}
		} else {
			if err = Dim.AddLabel(image, labels, tag); err != nil {
				return err
			}
		}

		if OverrideFlag {
			if dim.ListContains(imageTags, image) && image != tag {
				if err = Dim.Remove(image); err != nil {
					return err
				}
			}
		}

		if RemoteFlag {
			if err = Dim.Push(tag, authConfig); err != nil {
				return err
			}
		}

		return nil
	},
}

var (
	ImageFlag    string
	RemoteFlag   bool
	OverrideFlag bool
	PullFlag     bool
	DeleteFlag   bool
)

func init() {
	labelCommand.Flags().BoolVarP(&DeleteFlag, "delete", "d", false, "Delete the label")
	labelCommand.Flags().StringVarP(&ImageFlag, "tag", "t", "", "Tag the new labeled image")
	labelCommand.Flags().BoolVarP(&RemoteFlag, "remote", "r", false, "Delete the original image both locally and on the remote registry")
	labelCommand.Flags().BoolVarP(&OverrideFlag, "override", "o", false, "Delete the original image locally only")
	labelCommand.Flags().BoolVarP(&PullFlag, "pull", "p", false, "Pull the image before adding label to ensure label is added to latest version")
	RootCommand.AddCommand(labelCommand)
}
