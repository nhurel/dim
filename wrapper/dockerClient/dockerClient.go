package dockerClient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/builder"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/utils"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// Docker interface exposes the method used to interact with the docker daemon
type Docker interface {
	ImageBuild(parent string, buildLabels map[string]string, tag string) error
	Pull(image string) error
	Inspect(image string) (types.ImageInspect, error)
	Remove(image string) error
	Push(image string) error
}

// DockerClient implements Docker interface
type DockerClient struct {
	c        *client.Client
	Auth     *types.AuthConfig
	Insecure bool
}

// Client connects to the daemon and returns client object to interact with it
func (dc *DockerClient) Client() (*client.Client, error) {
	if dc.c == nil {
		var cli *client.Client
		var err error
		if cli, err = client.NewEnvClient(); err != nil {
			return nil, err
		}
		dc.c = cli
	}
	return dc.c, nil
}

// ImageBuild builds a new image
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
	var msg = &buildStream{}
	for dec.More() {
		dec.Decode(msg)
		logrus.Infoln(msg.Stream)
	}
	fmt.Println(msg.Stream)

	return nil
}

type buildStream struct {
	Stream string `json:"stream,omitempty"`
}

// Pull pulls an image from a registry
func (dc *DockerClient) Pull(image string) error {
	var c *client.Client
	var err error
	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return err
	}

	var resp io.ReadCloser

	var a string
	var n reference.Named
	if n, err = reference.ParseNamed(image); err != nil {
		return err
	}

	if a, err = dc.Authenticate(n.Hostname()); err != nil {
		return err
	}
	resp, err = c.ImagePull(context.Background(), image, types.ImagePullOptions{RegistryAuth: a})

	if resp != nil {
		defer resp.Close()
		dec := json.NewDecoder(resp)
		var msg = &pullStream{}
		for dec.More() {
			dec.Decode(msg)
			fmt.Println(msg.Status)
		}
	}

	return err
}

func (dc *DockerClient) Authenticate(registryUrl string) (string, error) {

	if dc.Auth == nil {
		dc.Auth = &types.AuthConfig{}
	}

	req, _ := http.NewRequest(http.MethodGet, utils.BuildURL(fmt.Sprintf("%s/v2/", registryUrl), dc.Insecure), &bytes.Buffer{})
	req.SetBasicAuth(dc.Auth.Username, dc.Auth.Password)
	logrus.WithFields(logrus.Fields{"URL": req.URL, "Login": dc.Auth.Username, "Password": dc.Auth.Password}).Debugln("Testing credentials")
	var resp *http.Response
	var err error

	for resp, err = http.DefaultClient.Do(req); (resp == nil || resp.StatusCode == http.StatusUnauthorized) && err == nil; {
		utils.ReadCredentials(dc.Auth)
		logrus.WithFields(logrus.Fields{"URL": req.URL, "Login": dc.Auth.Username, "Password": dc.Auth.Password}).Debugln("Testing credentials")
		req.SetBasicAuth(dc.Auth.Username, dc.Auth.Password)
		resp, err = http.DefaultClient.Do(req)
		if resp.StatusCode > http.StatusUnauthorized {
			e, _ := ioutil.ReadAll(resp.Body)
			return "", fmt.Errorf("Server error occured : %s", string(e))
		}
		resp.Body.Close()
	}
	if err != nil {
		return "", err
	}
	return encodeAuthToBase64(*dc.Auth)

}

type pullStream struct {
	Status string `json:"status,omitempty"`
}

// Push pushes an image to a registry
func (dc *DockerClient) Push(image string) error {
	logrus.WithField("image", image).Debugln("Pushing image")
	var c *client.Client
	var err error
	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return err
	}

	var a string
	var n reference.Named
	if n, err = reference.ParseNamed(image); err != nil {
		return err
	}

	if a, err = dc.Authenticate(n.Hostname()); err != nil {
		return err
	}
	var resp io.ReadCloser
	resp, err = c.ImagePush(context.Background(), image, types.ImagePushOptions{RegistryAuth: a})

	if resp != nil {
		defer resp.Close()
		fmt.Print("Pushing image...")
		dec := json.NewDecoder(resp)
		for dec.More() {
			dec.Decode(struct{}{})
			fmt.Print(".")
		}
		fmt.Println(".")
	}

	return err
}

func encodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
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

// Remove removes an image locally
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
