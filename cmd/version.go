package cmd

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/environment"
	"github.com/nhurel/dim/lib/registry"
	"github.com/spf13/cobra"
)

func newVersionCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	versionCommand := &cobra.Command{
		Use:   "version",
		Short: "Prints dim version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(c, ctx, cmd, args)
		},
	}
	rootCommand.AddCommand(versionCommand)
}

func runVersion(c *cli.Cli, ctx context.Context, cmd *cobra.Command, args []string) error {
	return PrintVersion(c, ctx)
}

// PrintVersion prints current dim version
func PrintVersion(c *cli.Cli, ctx context.Context) error {
	var err error
	if _, err = fmt.Fprintf(c.Out, "dim version : %s\n", environment.Get(ctx, environment.VersionKey)); err != nil {
		return err
	}
	if registryURL == "" {
		return nil
	}
	var authConfig *types.AuthConfig
	if username != "" || password != "" {
		authConfig = &types.AuthConfig{Username: username, Password: password}
	}

	var client dim.RegistryClient
	var infos *dim.Info

	client, err = registry.SilentNew(c, authConfig, registryURL)

	if err == nil {
		infos, err = client.ServerVersion()
	}

	if err != nil {
		fmt.Fprintf(c.Out, "N/A (%v)\n", err)
	} else {
		_, err = fmt.Fprintf(c.Out, "server version : %s\nserver uptime : %s\n", infos.Version, infos.Uptime)
	}

	return err
}
