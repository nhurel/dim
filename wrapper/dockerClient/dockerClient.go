package dockerClient

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/builder"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"strings"
)

type Docker interface {
	ImageBuild(parent string, buildLabels map[string]string, tag string) error
	Pull(image string) error
	Inspect(image string) (types.ImageInspect, error)
	Remove(image string) error
}

type DockerClient struct {
	c *client.Client
}

func (dc *DockerClient) Client() (*client.Client, error) {
	if dc.c == nil {
		if cli, err := client.NewEnvClient(); err != nil {
			return nil, err
		} else {
			dc.c = cli
		}
	}
	return dc.c, nil
}

func (dc *DockerClient) ImageBuild(parent string, buildLabels map[string]string, tag string) error {
	buildCtx, _, err := builder.GetContextFromReader(ioutil.NopCloser(strings.NewReader(fmt.Sprintf("FROM %s", parent))), "")
	// Setup an upload progress bar
	progressOutput := streamformatter.NewStreamFormatter().NewProgressOutput(&ioutils.NopWriter{}, true)

	var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	var c *client.Client
	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return err
	}

	var resp types.ImageBuildResponse

	if resp, err = c.ImageBuild(context.Background(), body, types.ImageBuildOptions{Labels: buildLabels, Tags: []string{tag}, ForceRemove: true}); err != nil {
		logrus.WithError(err).Fatalln("Error occured while building new image")
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var msg = &BuildStream{}
	for dec.More() {
		dec.Decode(msg)
		logrus.Infoln(msg.Stream)
	}
	fmt.Println(msg.Stream)

	return nil
}

type BuildStream struct {
	Stream string `json:"stream,omitempty"`
}

func (dc *DockerClient) Pull(image string) error {
	var c *client.Client
	var err error
	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return err
	}

	resp, err := c.ImagePull(context.Background(), image, types.ImagePullOptions{})

	if resp != nil {
		defer resp.Close()
		dec := json.NewDecoder(resp)
		var msg = &PullStream{}
		for dec.More() {
			dec.Decode(msg)
			fmt.Println(msg.Status)
		}
	}

	return err

}

type PullStream struct {
	Status string `json:"status,omitempty"`
}

// Inspect return all details of an image
func (dc *DockerClient) Inspect(image string) (types.ImageInspect, error) {
	var c *client.Client
	var err error
	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return types.ImageInspect{}, err
	}

	resp, _, err := c.ImageInspectWithRaw(context.Background(), image, false)

	return resp, err
}

func (dc *DockerClient) Remove(image string) error {
	logrus.WithField("image", image).Debugln("Entering Remove")
	var c *client.Client
	var err error
	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return err
	}

	resp, err := c.ImageRemove(context.Background(), image, types.ImageRemoveOptions{Force: false, PruneChildren: true})
	logrus.WithField("result", resp).Debugln("Remove done")
	if len(resp) > 0 {
		logrus.WithError(err).Debugln(resp)
		for _, r := range resp {
			if r.Deleted != "" {
				fmt.Printf("%s deleted\n", r.Deleted)
			}
			if r.Untagged != "" {
				fmt.Printf("%s untagged\n", r.Untagged)
			}
		}
	}

	return err
}
