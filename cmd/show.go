package cmd

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/utils/templates"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
	"os"
	"text/template"
)

var showCommand = &cobra.Command{
	Use: "show image",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("image name is missing")
		}
		image := args[0]

		var t string
		if templateFlag != "" {
			t = templateFlag
		} else {
			t = infoTpl
		}

		var tpl *template.Template
		var err error
		if tpl, err = templates.Parse(t); err != nil {
			return err
		}

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

			return client.PrintImageInfo(os.Stdout, parsedName, tpl)
		}

		return Dim.PrintImageInfo(os.Stdout, image, tpl)
	},
}

var templateFlag string

func init() {
	//TODO Add --output flag to write in a file
	showCommand.Flags().StringVarP(&templateFlag, "template", "t", "", "Template to use to display image info")
	showCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Show image from remote repository")
	RootCommand.AddCommand(showCommand)
}

const infoTpl = `
Name : {{range $i, $e := .RepoTags}} {{if eq $i  0}}{{$e}}{{end}}{{end}}
Id :  {{.ID}}
Labels:
{{range $k, $v := .Config.Labels}}{{$k}} = {{$v}}
{{end}}
Tags:
{{range $i, $e := .RepoTags}}{{$e}}
{{end}}
Ports :
{{range $k, $v := .Config.ExposedPorts}}{{$k}} = {{$v}}
{{end}}
Volumes:
{{range $k, $v := .Config.Volumes}}{{$k}} = {{$v}}
{{end}}
Env :
{{ range $i, $e := .Config.Env}} {{$e}}
{{end}}
Entrypoint : {{.Config.Entrypoint}}
Command : {{.Config.Cmd}}
`
