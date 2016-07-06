package cmd

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/wrapper/dockerClient"
	"github.com/spf13/cobra"
)

var RootCommand = &cobra.Command{
	Use:   "dim",
	Short: "Docker Image Management is a simple cli to manage docker images",
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
	},
}

var logLevel string

func init() {
	RootCommand.PersistentFlags().StringVarP(&logLevel, "log", "l", "warn", "Set log level")
}

var Dim = &dim.Dim{Docker: &dockerClient.DockerClient{}}

// GuessTag returns the tag to apply to the image to build
func guessTag(tagOption string, imageName string, imageTags []string, override bool) (string, error) {
	logrus.WithFields(logrus.Fields{"tagOption": tagOption, "imageName": imageName, "imageTags": imageTags, "override": override}).Debug("Entering guessTag")
	tag := tagOption
	if override && tag == "" {
		if !dim.ListContains(imageTags, imageName) {
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
