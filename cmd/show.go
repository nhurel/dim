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
	"os"
	"text/template"

	"context"

	"github.com/docker/docker/reference"
	"github.com/docker/docker/utils/templates"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/spf13/cobra"
)

func newShowCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	showCommand := &cobra.Command{
		Use:   "show IMAGE",
		Short: "Shows details about an image",
		Long: `Print the defails of a local image.
Use the -o flag to write the details into a flag instead of writing to stdout.
Use the -r flag to print the details of an image on the private registry (not present locally)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(c, args)
		},
	}

	showCommand.Flags().StringVarP(&templateFlag, "template", "t", "", "Template to use to display image info")
	showCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Show image from remote repository")
	showCommand.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to file instead of stdout")
	rootCommand.AddCommand(showCommand)
}

func runShow(c *cli.Cli, args []string) error {
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

	var output = c.Out
	if outputFlag != "" {
		if output, err = os.Create(outputFlag); err != nil {
			return fmt.Errorf("Failed to open file %s : %v", outputFlag, err)
		}
	}
	defer output.Close()

	if remoteFlag {
		var client dim.RegistryClient
		var err error
		var parsedName reference.Named
		if client, parsedName, err = connectRegistry(c, image); err != nil {
			return err
		}

		return client.PrintImageInfo(output, parsedName, tpl)
	}

	return Dim.PrintImageInfo(output, image, tpl)
}

var templateFlag, outputFlag string

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
