// Copyright 2016
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/reference"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
)

func newDeleteCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	deleteCommand := &cobra.Command{
		Use:   "delete IMAGE[:TAG]",
		Short: "Deletes an image",
		Long: `Deletes the image IMAGE locally.
If no TAG is specified, latest will be used
If flag -r is given the image is also deleted on the remote registry.`,
		Example: `dim delete ubuntu
dim delete -r ubuntu:xenial
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(c, args)
		},
	}

	deleteCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Delete the image both locally and on the remote registry")
	rootCommand.AddCommand(deleteCommand)
}

func runDelete(c *cli.Cli, args []string) error {
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
		var client dim.RegistryClient

		logrus.WithField("hostname", parsedName.Hostname()).Debugln("Connecting to registry")

		if client, err = registry.New(c, authConfig, utils.BuildURL(parsedName.Hostname(), insecure)); err != nil {
			return fmt.Errorf("Failed to connect to registry : %v", err)
		}

		return client.DeleteImage(parsedName)
	}

	return nil
}
