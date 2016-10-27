package cmd

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
)

var deleteCommand = &cobra.Command{
	Use:   "delete IMAGE",
	Short: "Deletes an image",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("image name missing")
		}

		image := args[0]

		Dim.Remove(image)

		if remoteFlag {

			var parsedName reference.Named
			var err error
			if parsedName, err = reference.ParseNamed(image); err != nil || parsedName.Hostname() == "" {
				return fmt.Errorf("Fail to parse the name to delete the image on a remote repository %v", err)
			}

			var authConfig *types.AuthConfig
			if username != "" || password != "" {
				authConfig = &types.AuthConfig{Username: username, Password: password}
			}
			var client registry.Client

			logrus.WithField("hostname", parsedName.Hostname()).Debugln("Connecting to registry")

			if client, err = registry.New(authConfig, utils.BuildURL(parsedName.Hostname(), insecure)); err != nil {
				return fmt.Errorf("Failed to connect to registry : %v", err)
			}

			return client.DeleteImage(parsedName)
		}

		return nil
	},
}

func init() {
	deleteCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Delete the image both locally and on the remote registry")
	RootCommand.AddCommand(deleteCommand)
}
