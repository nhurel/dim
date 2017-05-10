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

package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/registry"
	imageParser "github.com/docker/engine-api/types/reference"
	"github.com/nhurel/dim/cli"
	"github.com/nhurel/dim/lib"
	"golang.org/x/net/context"
)

// Client implements RegistryClient interface
type Client struct {
	client.Registry
	transport   http.RoundTripper
	registryURL string
}

var ctx = context.Background()

// New creates a registry client. Handles getting right credentials from user
func New(c *cli.Cli, registryAuth *types.AuthConfig, registryURL string) (*Client, error) {
	return tryNew(c, registryAuth, registryURL, true)
}

// SilentNew creates a registry client but doesn't prompt credentials when required
func SilentNew(c *cli.Cli, registryAuth *types.AuthConfig, registryURL string) (*Client, error) {
	return tryNew(c, registryAuth, registryURL, false)
}

func tryNew(c *cli.Cli, registryAuth *types.AuthConfig, registryURL string, prompt bool) (*Client, error) {
	var err error
	var reg client.Registry

	if registryURL == "" {
		return nil, fmt.Errorf("No registry URL given")
	}

	transport := http.DefaultTransport

	if registryAuth != nil {
		transport = registry.AuthTransport(transport, registryAuth, true)
	}

	if reg, err = client.NewRegistry(ctx, registryURL, transport); err != nil {
		return nil, err
	}

	repos := make([]string, 1)
	l := logrus.WithField("registry", registryURL)
	for _, err = reg.Repositories(ctx, repos, ""); err != nil && err != io.EOF; _, err = reg.Repositories(ctx, repos, "") {
		switch err.(type) {
		case *client.UnexpectedHTTPStatusError, *url.Error, *client.UnexpectedHTTPResponseError:
			return nil, fmt.Errorf("Failed to join the registry : %v", err)
		}
		if !prompt {
			break
		}
		l.Debugln("Prompting for credentials")
		if registryAuth == nil {
			registryAuth = &types.AuthConfig{}
		}
		cli.ReadCredentials(c, registryAuth)
		transport = registry.AuthTransport(transport, registryAuth, true)
		if reg, err = client.NewRegistry(ctx, registryURL, transport); err != nil {
			return nil, err
		}
	}

	logrus.WithField("auth", registryAuth).Debugln("Created transport")

	return &Client{reg, transport, registryURL}, nil
}

// NewRepository creates a Repository object to query the registry about a specific repository
func (c *Client) NewRepository(parsedName reference.Named) (dim.Repository, error) {
	logrus.WithField("name", parsedName).Debugln("Creating new repository")

	var repo distribution.Repository
	var err error
	if repo, err = client.NewRepository(ctx, parsedName, c.registryURL, c.transport); err != nil {
		return &Repository{}, err
	}

	return &Repository{Repository: repo, client: c}, nil
}

// Search runs a search against the registry, handling dim advanced querying option
func (c *Client) Search(query, advanced string, offset, maxResults int) (*dim.SearchResults, error) {
	q := strings.TrimSpace(query)
	a := strings.TrimSpace(advanced)
	var err error

	var resp *http.Response

	values := url.Values{}
	if a != "" {
		values.Set("a", a)
	}
	if q != "" {
		values.Set("q", q)
	}

	for _, field := range []string{"Name", "Tag", "FullName", "Labels", "Envs", "Volumes", "ExposedPorts", "Size", "Created"} {
		values.Add("f", field)
	}

	values.Set("offset", strconv.Itoa(offset))
	values.Set("maxResults", strconv.Itoa(maxResults))

	httpClient := http.Client{Transport: c.transport}

	endpoint := strings.TrimSuffix(c.registryURL, "/") + "/v1/search"
	if resp, err = httpClient.PostForm(endpoint, values); err != nil {
		return nil, fmt.Errorf("Failed to send request : %v", err)
	}
	defer resp.Body.Close()
	var b []byte
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		results := &dim.SearchResults{}
		if err := json.NewDecoder(resp.Body).Decode(results); err != nil {
			return nil, fmt.Errorf("Failed to parse response : %v", err)
		}

		return results, nil
	}

	b, _ = ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Server returned an error : %s", string(b))
}

// WalkRepositories walks through all repositories and send them in the given channel
func (c *Client) WalkRepositories() <-chan dim.Repository {
	return WalkRepositories(c)
}

// WalkRepositories walks through all repositories and send them in the given channel
func WalkRepositories(c dim.RegistryClient) <-chan dim.Repository {
	repositories := make(chan dim.Repository, 5)

	go func() {
		var err error
		defer close(repositories)
		var n int
		registries := make([]string, 20)
		last := ""
		for stop := false; !stop; {

			if n, err = c.Repositories(nil, registries, last); err != nil && err != io.EOF {
				logrus.WithField("n", n).WithError(err).Errorln("Failed to get repostories")
				continue
			}

			stop = (err == io.EOF)

			for i := 0; i < n; i++ {
				last = registries[i]

				var parsedName reference.Named

				l := logrus.WithField("repository", last)
				l.Infoln("Indexing repository")
				if parsedName, err = reference.ParseNamed(last); err != nil {
					logrus.WithError(err).WithField("name", last).Errorln("Failed to parse repository name")
					continue
				}

				var repository dim.Repository

				if repository, err = c.NewRepository(parsedName); err != nil {
					logrus.WithError(err).WithField("name", last).Errorln("Failed to fetch repository info")
					continue
				}
				repositories <- repository
			}
		}

	}()
	return repositories

}

// PrintImageInfo prints the info about an image available on the remote registry
func (c *Client) PrintImageInfo(w io.Writer, parsedName reference.Named, tpl *template.Template) error {
	var repository dim.Repository
	var err error
	name, _ := reference.ParseNamed(parsedName.Name()[strings.Index(parsedName.Name(), "/")+1:])
	if repository, err = c.NewRepository(name); err != nil {
		logrus.WithError(err).Errorln("Failed to fetch repository info")
		return err
	}

	tag := ParseTag(parsedName)

	var image *dim.RegistryImage
	if image, err = repository.Image(tag); err != nil {
		logrus.WithError(err).Errorln("Failed to fetch image info")
		return err
	}

	info := &types.ImageInspect{
		RepoTags: []string{image.Tag},
		ID:       image.ImageID(),
		Config:   image.Config,
	}

	return tpl.Execute(w, info)
}

// DeleteImage deletes the image on the remote registry
func (c *Client) DeleteImage(parsedName reference.Named) error {
	logrus.WithField("parsedName", parsedName.String()).Debugln("Entering DeleteImage")
	var repo dim.Repository
	var err error
	name, _ := reference.ParseNamed(parsedName.Name()[strings.Index(parsedName.Name(), "/")+1:])
	if repo, err = c.NewRepository(name); err != nil {
		return err
	}

	tag := imageParser.GetTagFromNamedRef(parsedName)

	if tag == "" {
		tag = "latest"
	}

	logrus.Debugln("Deleting image")
	if err = repo.DeleteImage(tag); err != nil {
		logrus.WithError(err).Errorln("Failed to delete image on the remote registry")
		return err
	}

	return nil
}

// ServerVersion read dim server version information
func (c *Client) ServerVersion() (*dim.Info, error) {

	var resp *http.Response
	var err error
	httpClient := http.Client{Transport: c.transport}

	endpoint := strings.TrimSuffix(c.registryURL, "/") + "/dim/version"
	if resp, err = httpClient.Get(endpoint); err != nil {
		return nil, fmt.Errorf("Failed to send request : %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		infos := &dim.Info{}
		if err := json.NewDecoder(resp.Body).Decode(infos); err != nil {
			return nil, fmt.Errorf("Failed to parse response : %v", err)
		}

		return infos, nil
	}

	return nil, fmt.Errorf("Server returned an error : %s", resp.Status)
}

// ParseTag returns the tag corresponding to the given image name
func ParseTag(name reference.Named) string {
	var tag string
	switch parsedName := name.(type) {
	case reference.NamedTagged:
		tag = parsedName.Tag()
	default:
		tag = "latest"
	}
	return tag
}
