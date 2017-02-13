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

	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib/utils"
	"github.com/spf13/cobra"
)

func newLabelCommand(c *cli.Cli, rootCommand *cobra.Command, ctx context.Context) {
	labelCommand := &cobra.Command{
		Use:   "label [--delete] IMAGE[:TAG] LABEL_KEY[=LABEL_VALUE]...",
		Short: "Add / Remove a label to a given image",
		Long: `Add label to the image IMAGE. If no tag is given, latest will be used.
Multiple labels can be given at once, separated by a space.
To delete a tag, pass the --delete flag.
`,
		Example: `dim label ubuntu:xenial os=ubuntu version=xenial
dim label --delete ubuntu:xenial os version
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("Missing argument. See help")
			}
			image := args[0]
			labels := args[1:]

			var imageTags []string
			var tag string
			var err error

			if pullFlag {
				if err = Dim.Pull(image); err != nil {
					return err
				}
			}

			if _, imageTags, err = Dim.GetImageInfo(image); err != nil {
				return err
			}

			if tag, err = guessTag(imageFlag, image, imageTags, overrideFlag); err != nil {
				return err
			}

			if deleteFlag {
				if err = Dim.RemoveLabel(image, labels, tag); err != nil {
					return err
				}
			} else {
				if err = Dim.AddLabel(image, labels, tag); err != nil {
					return err
				}
			}

			if overrideFlag {
				if utils.ListContains(imageTags, image) && image != tag {
					if err = Dim.Remove(image); err != nil {
						return err
					}
				}
			}

			if remoteFlag {
				if err = Dim.Push(tag); err != nil {
					return err
				}
			}

			return nil
		},
	}

	labelCommand.Flags().BoolVarP(&deleteFlag, "delete", "d", false, "Delete the label")
	labelCommand.Flags().StringVarP(&imageFlag, "tag", "t", "", "Tag the new labeled image")
	labelCommand.Flags().BoolVarP(&remoteFlag, "remote", "r", false, "Tag or Delete the original image both locally and on the remote registry")
	labelCommand.Flags().BoolVarP(&overrideFlag, "override", "o", false, "Delete the original image locally only")
	labelCommand.Flags().BoolVarP(&pullFlag, "pull", "p", false, "Pull the image before adding label to ensure label is added to latest version")
	rootCommand.AddCommand(labelCommand)
}

var (
	imageFlag    string
	remoteFlag   bool
	overrideFlag bool
	pullFlag     bool
	deleteFlag   bool
)
