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
	"strings"

	"net/url"

	"context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/reference"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/index"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
	"github.com/nhurel/dim/server"
	"github.com/nhurel/dim/wrapper/dockerClient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand builds the whole command list
func NewRootCommand(cli *cli.Cli, ctx context.Context) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:          "dim",
		Short:        "Docker Image Management is a simple cli to manage docker images",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			switch logLevel {
			case "debug":
				logrus.SetLevel(logrus.DebugLevel)
			case "info":
				logrus.SetLevel(logrus.InfoLevel)
			case "warn":
				logrus.SetLevel(logrus.WarnLevel)
			case "error":
				logrus.SetLevel(logrus.ErrorLevel)
			case "fatal":
				logrus.SetLevel(logrus.FatalLevel)
			}

			u := viper.GetString("registry-url")
			registryURL = utils.BuildURL(u, insecure)

			username = viper.GetString("registry-user")
			password = viper.GetString("registry-password")

			var authConfig *types.AuthConfig
			if username != "" || password != "" {
				authConfig = &types.AuthConfig{Username: username, Password: password}
			}

			Dim = &dim.Dim{Docker: &dockerClient.DockerClient{Cli: cli, Auth: authConfig, Insecure: insecure}}
		},
		BashCompletionFunction: bashCompletionFunc,
	}

	rootCommand.PersistentFlags().StringVarP(&logLevel, "log", "l", "warn", "Set log level")
	rootCommand.PersistentFlags().String("registry-url", "", "Registry URL or hostname")
	rootCommand.PersistentFlags().String("registry-user", "", "Registry username")
	rootCommand.PersistentFlags().String("registry-password", "", "Registry password")
	rootCommand.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Connect to registry through http instead of https")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.BindPFlag("registry-url", rootCommand.PersistentFlags().Lookup("registry-url"))
	viper.BindPFlag("registry-user", rootCommand.PersistentFlags().Lookup("registry-user"))
	viper.BindPFlag("registry-password", rootCommand.PersistentFlags().Lookup("registry-password"))
	viper.BindEnv("registry-url")
	viper.BindEnv("registry-user")
	viper.BindEnv("registry-password")

	viper.SetConfigType("yaml")
	viper.SetConfigName("dim")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.dim")
	if err := viper.ReadInConfig(); err != nil {
		switch err.(type) {
		case *os.PathError, viper.ConfigFileNotFoundError:
			logrus.WithError(err).Debugln("No config file found")
		default:
			logrus.WithError(err).Fatalln("Failed to read config file")
		}
	}

	newDeleteCommand(cli, rootCommand, ctx)
	newGenBashCompletionCommand(cli, rootCommand, ctx)
	newLabelCommand(cli, rootCommand, ctx)
	newSearchCommand(cli, rootCommand, ctx)
	newServerCommand(cli, rootCommand, ctx)
	newShowCommand(cli, rootCommand, ctx)
	newVersionCommand(cli, rootCommand, ctx)
	newHooktestCommand(cli, rootCommand, ctx)
	newGenPasswdCommand(cli, rootCommand, ctx)

	return rootCommand
}

var logLevel string
var (
	registryURL string
	username    string
	password    string
	insecure    bool
)

// Dim instance has a dockerClient object to interact with docker daemon
var Dim *dim.Dim

func parseName(image, registryURL string) (reference.Named, error) {
	var parsedName reference.Named
	var err error
	if parsedName, err = reference.ParseNamed(image); err != nil {
		return nil, fmt.Errorf("Failed to parse the name of the remote repository image %s : %v", image, err)
	}
	if parsedName.Hostname() == reference.DefaultHostname && !strings.HasPrefix(image, reference.DefaultHostname) {
		fullURL, err := url.Parse(registryURL)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse registry URL : %v", err)
		}

		logrus.WithField("registryUrl", fullURL.Host).Infoln("Adding registry URL in image name")
		if parsedName, err = reference.ParseNamed(fmt.Sprintf("%s/%s", fullURL.Host, image)); err != nil {
			return nil, fmt.Errorf("Failed to parse the name of the remote repository image : %v", err)
		}
	}
	return parsedName, nil
}

// guessTag returns the tag to apply to the image to build
func guessTag(tagOption string, imageName string, imageTags []string, override bool) (string, error) {
	logrus.WithFields(logrus.Fields{"tagOption": tagOption, "imageName": imageName, "imageTags": imageTags, "override": override}).Debug("Entering guessTag")
	tag := tagOption
	if override && tag == "" {
		if !utils.ListContains(imageTags, imageName) {
			if len(imageTags) > 0 {
				tag = imageTags[0]
			} else {
				return "", fmt.Errorf("Cannot override image with no tag. Use --tag option instead")
			}
		} else {
			tag = imageName
		}
	}
	return tag, nil
}

func connectRegistry(c *cli.Cli, image string) (client dim.RegistryClient, parsedName reference.Named, err error) {
	if parsedName, err = parseName(image, registryURL); err != nil {
		return
	}

	var authConfig *types.AuthConfig
	if username != "" || password != "" {
		authConfig = &types.AuthConfig{Username: username, Password: password}
	}

	logrus.WithField("hostname", parsedName.Hostname()).Debugln("Connecting to registry")

	if client, err = registry.New(c, authConfig, utils.BuildURL(parsedName.Hostname(), insecure)); err != nil {
		err = fmt.Errorf("Failed to connect to registry : %v", err)
		return
	}
	return
}

func readConfigHooks(hookFns map[string]interface{}) (*index.Config, error) {
	cfg := &index.Config{}

	hooks := make([]*index.Hook, 0, 10)
	if err := viper.UnmarshalKey("index.hooks", &hooks); err != nil {
		return nil, err
	}

	cfg.Hooks = hooks

	for n, f := range hookFns {
		cfg.RegisterFunction(n, f)
	}

	if err := cfg.ParseHooks(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func readServerConfig() (*server.Config, error) {
	cfg := &server.Config{Port: port}
	auths := make([]*server.Authorization, 0, 10)
	if err := viper.UnmarshalKey("server.security", &auths); err != nil {
		return nil, err
	}

	for _, auth := range auths {
		if err := auth.CompilePath(); err != nil {
			return nil, err
		}
	}

	cfg.Authorizations = auths
	return cfg, nil
}

const (
	bashCompletionFunc = `
__custom_func() {
	case ${last_command} in
		dim_show | dim_delete | dim_label | dim_hooktest)
			__docker_complete_image_repos_and_tags
			return
			;;
		*)
			;;
	esac
}

__docker_q() {
        docker ${host:+-H "$host"} ${config:+--config "$config"} 2>/dev/null "$@"
}

__docker_images() {
        local images_args=""

        case "$DOCKER_COMPLETION_SHOW_IMAGE_IDS" in
                all)
                        images_args="--no-trunc -a"
                        ;;
                non-intermediate)
                        images_args="--no-trunc"
                        ;;
        esac

        local repo_print_command
        if [ "${DOCKER_COMPLETION_SHOW_TAGS:-yes}" = "yes" ]; then
                repo_print_command='print $1; print $1":"$2'
        else
                repo_print_command='print $1'
        fi

        local awk_script
        case "$DOCKER_COMPLETION_SHOW_IMAGE_IDS" in
                all|non-intermediate)
                        awk_script='NR>1 { print $3; if ($1 != "<none>") { '"$repo_print_command"' } }'
                        ;;
                none|*)
                        awk_script='NR>1 && $1 != "<none>" { '"$repo_print_command"' }'
                        ;;
        esac

        __docker_q images $images_args | awk "$awk_script" | grep -v '<none>$'
}

__docker_complete_image_repos_and_tags() {
        local reposAndTags="$(__docker_q images | awk 'NR>1 && $1 != "<none>" { print $1; print $1":"$2 }')"
        COMPREPLY=( $(compgen -W "$reposAndTags" -- "$cur") )
        __ltrim_colon_completions "$cur"
}

	`
)
