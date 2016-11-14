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
	Use:   "delete IMAGE[:TAG]",
	Short: "Deletes an image",
	Long: `Deletes the image IMAGE locally.
If no TAG is specified, latest will be used
If flag -r is given the image is also deleted on the remote registry.`,
	Example: `dim delete ubuntu
dim delete -r ubuntu:xenial
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("image name missing")
		}

		image := args[0]

		Dim.Remove(image)

		if remoteFlag {

			var parsedName reference.Named
			var err error
			if parsedName, err = parseName(image, registryURL); err != nil {
				return err
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
