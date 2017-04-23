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
	"errors"
	"fmt"
	"strings"

	"strconv"

	"time"

	"context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
)

func newSearchCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	searchCommand := &cobra.Command{
		Use:   "search QUERY",
		Short: "Run a search against a private registry",
		Long: `Search an image on the private registry.
By default the provided query is searched in the names and tags of the images on the registry.
Using -a flag, you can run advanced queries and search in the labels and volumes too.`,
		Example: `# Find the images with label os=ubuntu
dim search -a Label.os:ubuntu
# Find the images having a label 'os'
dim search -a Labels:os

With the -a flag, you can also use the +/- operator to combine your clauses :
dim search -a +Label.os:ubuntu -Label.version=xenial`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(c, args)
		},
	}

	searchCommand.Flags().BoolVarP(&advancedFlag, "advanced", "a", false, "Runs complex query")
	searchCommand.Flags().IntVar(&paginationFlag, "bulk-size", 15, "Number of restuls to fetch at a time")
	searchCommand.Flags().BoolVarP(&unlimitedFlag, "unlimited", "u", false, "Prints all results at once")
	searchCommand.Flags().IntVarP(&widthFlag, "width", "W", 150, "Column width")
	searchCommand.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Print only image fullname")
	searchCommand.Flags().StringVarP(&templateFlag, "template", "t", "", "Template to use to display image info")
	rootCommand.AddCommand(searchCommand)
}

func runSearch(c *cli.Cli, args []string) error {
	if len(args) == 0 {
		return errors.New("query is missing")
	}
	query := args[0]

	var authConfig *types.AuthConfig
	if username != "" || password != "" {
		authConfig = &types.AuthConfig{Username: username, Password: password}
	}

	var client dim.RegistryClient
	var err error

	logrus.WithField("url", registryURL).Debugln("Connecting to registry")

	if client, err = registry.New(c, authConfig, registryURL); err != nil {
		return fmt.Errorf("Failed to connect to registry : %v", err)
	}

	var q, a string
	if advancedFlag {
		a = query
	} else {
		q = query
	}

	var results *dim.SearchResults
	if results, err = client.Search(q, a, 0, paginationFlag); err != nil {
		return fmt.Errorf("Failed to search images : %v", err)
	}

	if results.NumResults > 0 {
		fmt.Fprintf(c.Err, "%d results found :\n", results.NumResults)
		var printer cli.Printer
		template := guessTemplate(quietFlag, templateFlag)
		switch template {
		case "":
			printer = cli.NewTabPrinter(c.Out, c.In, cli.WithWidth(widthFlag))
			printer.(*cli.TabPrinter).Append([]string{"Name", "Tag", "Created", "Labels", "Volumes", "Ports"})
		default:
			printer = cli.NewTemplatePrinter(c.Out, c.In, template)
		}

		for _, r := range results.Results {
			printAppend(printer, r)
		}

		if err := printer.PrintAll(false); err != nil {
			return err
		}
		for fetched := len(results.Results); results.NumResults > fetched; {
			if unlimitedFlag {
				c.Out.Write([]byte("\n"))
			}
			if results, err = client.Search(q, a, fetched, paginationFlag); err != nil {
				return fmt.Errorf("Failed to search images : %v", err)
			}
			for _, r := range results.Results {
				printAppend(printer, r)
			}

			if err := printer.PrintAll(!unlimitedFlag); err != nil {
				return err
			}
			fetched += len(results.Results)
		}
		fmt.Println()
	} else {
		fmt.Fprintln(c.Err, "No result found")
	}

	return nil
}

func intToStringSlice(iSlice []int) []string {
	result := make([]string, len(iSlice))
	for ind, i := range iSlice {
		result[ind] = strconv.Itoa(i)
	}
	return result
}

func printAppend(printer cli.Printer, r dim.SearchResult) {
	if p, ok := printer.(*cli.TabPrinter); ok {
		p.Append([]string{r.Name, r.Tag, utils.ParseDuration(time.Since(r.Created)), utils.FlatMap(r.Label), strings.Join(r.Volumes, ","), strings.Join(intToStringSlice(r.ExposedPorts), ",")})
	} else if p, ok := printer.(*cli.TemplatePrinter); ok {
		p.Append(r)
	}
}

func guessTemplate(quiet bool, tpl string) string {
	if quiet {
		return "{{.FullName}}"
	}
	return tpl
}

var (
	advancedFlag   bool
	paginationFlag int
	widthFlag      int
	unlimitedFlag  bool
	quietFlag      bool
)
