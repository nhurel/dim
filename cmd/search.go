package cmd

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/registry"
	"github.com/spf13/cobra"
)

var searchCommand = &cobra.Command{
	Use: "search query",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("query is missing")
		}
		query := args[0]

		var authConfig *types.AuthConfig
		if username != "" || password != "" {
			authConfig = &types.AuthConfig{Username: username, Password: password}
		}

		var client registry.Client
		var err error

		logrus.WithField("url", url).Debugln("Connecting to registry")

		if client, err = registry.New(authConfig, url); err != nil {
			return fmt.Errorf("Failed to connect to registry : %v", err)
		}

		var q, a string
		if advancedFlag {
			a = query
		} else {
			q = query
		}

		return client.Search(q, a)
	},
}

var advancedFlag bool

func init() {
	searchCommand.Flags().BoolVarP(&advancedFlag, "advanced", "a", false, "Runs complex query")
	RootCommand.AddCommand(searchCommand)
}
