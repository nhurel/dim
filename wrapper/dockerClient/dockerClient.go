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

package dockerClient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"time"

	"github.com/Sirupsen/logrus"
	apitypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib/utils"
	"golang.org/x/net/context"
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
	Cli      *cli.Cli
	Auth     *apitypes.AuthConfig
	Insecure bool
}

// ErrDockerHubAuthenticationNotSupported is thrown when trying to authenticate on a non private repository
var ErrDockerHubAuthenticationNotSupported = fmt.Errorf("Authentication on docker hub not supported")

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
	var err error
	var c *client.Client

	if c, err = dc.Client(); err != nil {
		logrus.WithError(err).Fatalln("Error occured while connecting to docker daemon")
		return err
	}

	tempName := fmt.Sprintf("dim_%s", time.Now().Format("20060102150405.000"))
	var created types.ContainerCreateResponse
	if created, err = c.ContainerCreate(context.Background(), &container.Config{Image: parent}, nil, nil, tempName); err != nil {
		logrus.WithField("image", parent).WithError(err).Fatalln("Failed to create temp container")
		return err
	}

	if _, err = c.ContainerCommit(context.Background(), created.ID, types.ContainerCommitOptions{Changes: []string{changeLabels(buildLabels)}, Reference: tag}); err != nil {
		logrus.WithError(err).Fatalln("Failed to commit new labels")
		return err
	}

	if err := c.ContainerRemove(context.Background(), created.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
		logrus.WithField("name", tempName).WithField("id", created.ID).WithError(err).Warnln("Failed to delete temp container")
	}

	return nil
}

func changeLabels(m map[string]string) string {
	entries := make([]string, 0, len(m))
	for k, v := range m {
		entries = append(entries, fmt.Sprintf("%s=\"%s\"", k, v))
	}
	return fmt.Sprintf("LABEL %s", strings.Join(entries, " "))
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
		if err == ErrDockerHubAuthenticationNotSupported {
			logrus.WithError(err).Warnln("Pulling image from docker hub as unauthenticated user")
		} else {
			return err
		}
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

// Authenticate prompts the user his credentials until it can connect to the registry
func (dc *DockerClient) Authenticate(registryURL string) (string, error) {

	if registryURL == reference.DefaultHostname {
		return "", ErrDockerHubAuthenticationNotSupported
	}

	if dc.Auth == nil {
		dc.Auth = &apitypes.AuthConfig{}
	}

	req, _ := http.NewRequest(http.MethodGet, utils.BuildURL(fmt.Sprintf("%s/v2/", registryURL), dc.Insecure), &bytes.Buffer{})
	req.SetBasicAuth(dc.Auth.Username, dc.Auth.Password)
	logrus.WithFields(logrus.Fields{"URL": req.URL, "Login": dc.Auth.Username, "Password": dc.Auth.Password}).Debugln("Testing credentials")
	var resp *http.Response
	var err error

	for resp, err = http.DefaultClient.Do(req); (resp == nil || resp.StatusCode == http.StatusUnauthorized) && err == nil; {
		cli.ReadCredentials(dc.Cli, dc.Auth)
		logrus.WithFields(logrus.Fields{"URL": req.URL, "Login": dc.Auth.Username, "Password": dc.Auth.Password}).Debugln("Testing credentials")
		req.SetBasicAuth(dc.Auth.Username, dc.Auth.Password)
		resp, err = http.DefaultClient.Do(req)
		if resp.StatusCode > http.StatusUnauthorized {
			e, _ := ioutil.ReadAll(resp.Body)
			return "", fmt.Errorf("Server error occured : %s", string(e))
		}
		resp.Body.Close()
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%s not a private registry", req.URL)
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

func encodeAuthToBase64(authConfig apitypes.AuthConfig) (string, error) {
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

	resp, _, err := c.ImageInspectWithRaw(context.Background(), image)

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
