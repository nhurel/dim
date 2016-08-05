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
	Use: "search",
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		if query == "" {
			return errors.New("Search is mandatory")
		}

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
		if AdvancedFlag {
			a = query
		} else {
			q = query
		}

		return client.Search(q, a)
	},
}

var AdvancedFlag bool

func init() {
	searchCommand.Flags().BoolVarP(&AdvancedFlag, "advanced", "a", false, "Runs complex query")
	RootCommand.AddCommand(searchCommand)
}
