package cmd

import (
	"context"
	"fmt"

	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
)

func newGenPasswdCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	genpasswdCommand := &cobra.Command{
		Use:   "genpasswd QUERY",
		Short: "Encode a password in sha256",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenPasswd(c, args)
		},
	}

	rootCommand.AddCommand(genpasswdCommand)
}

func runGenPasswd(c *cli.Cli, args []string) error {
	var password string
	if len(args) > 0 {
		password = args[0]
	} else {
		for password == "" {
			fmt.Fprint(c.Out, "Password :")
			cli.ReadPassword(&password)
		}
	}
	fmt.Fprintln(c.Out, utils.Sha256(password))
	return nil
}
