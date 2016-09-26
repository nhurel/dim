package cmd

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/utils"
	"github.com/nhurel/dim/wrapper/dockerClient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

// RootCommand is the main cobraCommand all other commands are attached to
var RootCommand = &cobra.Command{
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
		url = utils.BuildURL(u, insecure)

		username = viper.GetString("registry-user")
		password = viper.GetString("registry-password")

		var authConfig *types.AuthConfig
		if username != "" || password != "" {
			authConfig = &types.AuthConfig{Username: username, Password: password}
		}

		Dim = &dim.Dim{Docker: &dockerClient.DockerClient{Auth: authConfig, Insecure: insecure}}
	},
}

var logLevel string
var (
	url      string
	username string
	password string
	insecure bool
)

func init() {
	RootCommand.PersistentFlags().StringVarP(&logLevel, "log", "l", "warn", "Set log level")
	RootCommand.PersistentFlags().String("registry-url", "", "Registry URL or hostname")
	RootCommand.PersistentFlags().String("registry-user", "", "Registry username")
	RootCommand.PersistentFlags().String("registry-password", "", "Registry password")
	RootCommand.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Connect ot registry through http instead of https")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.BindPFlag("registry-url", RootCommand.PersistentFlags().Lookup("registry-url"))
	viper.BindPFlag("registry-user", RootCommand.PersistentFlags().Lookup("registry-user"))
	viper.BindPFlag("registry-password", RootCommand.PersistentFlags().Lookup("registry-password"))
	viper.BindEnv("registry-url")
	viper.BindEnv("registry-user")
	viper.BindEnv("registry-password")

	viper.SetConfigType("yaml")
	viper.SetConfigName("dim")
	viper.AddConfigPath("$HOME/.dim")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		switch err.(type) {
		case *os.PathError:
			logrus.WithError(err).Debugln("No config file found")
		default:
			logrus.WithError(err).Fatalln("Failed to read config file")
		}
	}

}

// Dim instance has a dockerClient object to interact with docker daemon
var Dim *dim.Dim

// GuessTag returns the tag to apply to the image to build
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
