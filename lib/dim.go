package dim

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/wrapper/dockerClient"
	"io"
	"strings"
	"text/template"
)

// Dim is the client type that handle all client side interaction with docker daemon
type Dim struct {
	Docker dockerClient.Docker
}

// AddLabel applies the given labels to the parent and tag the new created image with the given tag
func (d *Dim) AddLabel(parent string, labels []string, tag string) error {
	logrus.WithFields(logrus.Fields{"parent": parent, "labels": labels}).Debugln("Entering AddLabel")

	buildLabels := make(map[string]string)

	if len(labels) == 0 {
		return fmt.Errorf("No label provided")
	}

	for _, l := range labels {
		//TODO Use regexp to allow '=' in label value
		kv := strings.Split(l, "=")
		if len(kv) != 2 || kv[1] == "" {
			logrus.WithField("label", l).Infoln("Failed to parse given label")
			return fmt.Errorf("Failed to parse given label %s", l)
		}
		buildLabels[kv[0]] = kv[1]
	}

	var actualLabels map[string]string
	var err error
	if actualLabels, err = d.GetImageLabels(parent); err != nil {
		return err
	}

	if MapMatchesAll(actualLabels, buildLabels) {
		return fmt.Errorf("Image %s already contains the label(s) you want to set", parent)
	}

	return d.Docker.ImageBuild(parent, buildLabels, tag)
}

// Pull pulls the given image (must be fully qualified)
func (d *Dim) Pull(image string) error {
	return d.Docker.Pull(image)
}

// GetImageInfo returns the imageID and tags of an image
func (d *Dim) GetImageInfo(image string) (string, []string, error) {
	i, err := d.Docker.Inspect(image)
	if err != nil {
		return "", nil, err
	}

	return i.ID, i.RepoTags, err
}

//GetImageLabels returns all the labels of a given image
func (d *Dim) GetImageLabels(image string) (map[string]string, error) {
	i, err := d.Docker.Inspect(image)
	if err != nil {
		return nil, err
	}

	return i.ContainerConfig.Labels, err
}

//Remove deletes an image locally
func (d *Dim) Remove(image string) error {
	return d.Docker.Remove(image)
}

// Push pushes an image to a registry
func (d *Dim) Push(image string, auth *types.AuthConfig) error {
	return d.Docker.Push(image, auth)
}

// RemoveLabel clear the given labels to image parent and applies the fiven tag to the newly bulit image. Labels cannot be deleted so their value is only reset to an empty string
// TODO Implement remove labels by pattern
func (d *Dim) RemoveLabel(parent string, labels []string, tag string) error {
	logrus.WithFields(logrus.Fields{"parent": parent, "labels": labels}).Debugln("Entering RemoveLabel")

	buildLabels := make(map[string]string)

	if len(labels) == 0 {
		return fmt.Errorf("No label provided")
	}

	actualLabels, err := d.GetImageLabels(parent)
	if err != nil {
		return err
	}

	for _, l := range labels {
		if strings.Contains(l, "=") {
			logrus.WithField("label", l).Infoln("Failed to parse given label")
			return fmt.Errorf("Failed to parse given label %s", l)
		}
		if actualLabels[l] != "" {
			buildLabels[l] = ""
		}

	}

	if len(buildLabels) == 0 {
		return fmt.Errorf("Image %s has none of the given label(s) you want to clear", parent)
	}

	return d.Docker.ImageBuild(parent, buildLabels, tag)
}

// PrintImageInfo writes image information to the writer
func (d *Dim) PrintImageInfo(w io.Writer, image string, tpl *template.Template) error {
	var err error
	var infos types.ImageInspect

	if infos, err = d.Docker.Inspect(image); err != nil {
		return err
	}

	return tpl.Execute(w, infos)

}
