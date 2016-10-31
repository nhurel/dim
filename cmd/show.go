package cmd

import (
	"fmt"
	"os"
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/utils/templates"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
)

var showCommand = &cobra.Command{
	Use:   "show IMAGE",
	Short: "Shows details about an image",
	Long: `Print the defails of a local image.
Use the -o flag to write the details into a flag instead of writing to stdout.
Use the -r flag to print the details of an image on the private registry (not present locally)`,
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

		var output = os.Stdout
		if outputFlag != "" {
			if output, err = os.Create(outputFlag); err != nil {
				return fmt.Errorf("Failed to open file %s : %v", outputFlag, err)
			}
		}
		defer output.Close()

		if remoteFlag {
			var parsedName reference.Named
			if parsedName, err = reference.ParseNamed(image); err != nil || parsedName.Hostname() == "" {
				return fmt.Errorf("Failed to parse the name to delete the image on a remote repository %v", err)
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

			return client.PrintImageInfo(output, parsedName, tpl)
		}

		return Dim.PrintImageInfo(output, image, tpl)
	},
}

var templateFlag, outputFlag string

func init() {
	showCommand.Flags().StringVarP(&templateFlag, "template", "t", "", "Template to use to display image info")
	showCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Show image from remote repository")
	showCommand.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to file instead of stdout")
	RootCommand.AddCommand(showCommand)
}

const infoTpl = `Name : {{range $i, $e := .RepoTags}} {{if eq $i  0}}{{$e}}{{end}}{{end}}
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
