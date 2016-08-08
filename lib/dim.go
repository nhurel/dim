package dim

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/utils/templates"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/wrapper/dockerClient"
	"io"
	"strings"
	"text/template"
)

type Dim struct {
	Docker dockerClient.Docker
}

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

	if actualLabels, err := d.GetImageLabels(parent); err != nil {
		return err
	} else {
		if MapMatchesAll(actualLabels, buildLabels) {
			return fmt.Errorf("Image %s already contains the label(s) you want to set", parent)
		}
	}

	return d.Docker.ImageBuild(parent, buildLabels, tag)
}

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

func (d *Dim) GetImageLabels(image string) (map[string]string, error) {
	i, err := d.Docker.Inspect(image)
	if err != nil {
		return nil, err
	}

	return i.ContainerConfig.Labels, err
}

func (d *Dim) Remove(image string) error {
	return d.Docker.Remove(image)
}

func (d *Dim) Push(image string, auth *types.AuthConfig) error {
	return d.Docker.Push(image, auth)
}

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

// PrintImageInfo writes image information to the writer
func (d *Dim) PrintImageInfo(w io.Writer, image, optionTpl string) error {
	var err error
	var infos types.ImageInspect

	var tpl *template.Template

	if infos, err = d.Docker.Inspect(image); err != nil {
		return err
	}

	var t string
	if optionTpl != "" {
		t = optionTpl
	} else {
		t = infoTpl
	}

	if tpl, err = templates.Parse(t); err != nil {
		return err
	}

	err = tpl.Execute(w, infos)

	return err
}
