package registry

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/registry"
	"github.com/docker/engine-api/types"
	imageParser "github.com/docker/engine-api/types/reference"
	"github.com/nhurel/dim/lib/utils"
	t "github.com/nhurel/dim/types"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
)

// Client defines method to interact with a registry
type Client interface {
	client.Registry
	NewRepository(parsedName reference.Named) (Repository, error)
	Search(query, advanced string, offset, maxResults int) (*t.SearchResults, error)
	WalkRepositories(repositories chan<- Repository) error
	PrintImageInfo(out io.Writer, parsedName reference.Named, tpl *template.Template) error
	DeleteImage(parsedName reference.Named) error
}

// RegistryClient implements Client interface
type registryClient struct {
	client.Registry
	transport   http.RoundTripper
	registryURL string
}

var ctx = context.Background()

// New creates a registry client. Handles getting right credentials from user
func New(registryAuth *types.AuthConfig, registryURL string) (Client, error) {
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
	for _, err = reg.Repositories(ctx, repos, ""); err != nil && err != io.EOF; _, err = reg.Repositories(ctx, repos, "") {
		logrus.Debugln("Prompting for credentials")
		if registryAuth == nil {
			registryAuth = &types.AuthConfig{}
		}
		utils.ReadCredentials(registryAuth)
		transport = registry.AuthTransport(transport, registryAuth, true)
		if reg, err = client.NewRegistry(ctx, registryURL, transport); err != nil {
			return nil, err
		}
	}

	return &registryClient{reg, transport, registryURL}, nil
}

// NewRepository creates a Repository object to query the registry about a specific repository
func (c *registryClient) NewRepository(parsedName reference.Named) (Repository, error) {
	logrus.WithField("name", parsedName).Debugln("Creating new repository")

	var repo distribution.Repository
	var err error
	if repo, err = client.NewRepository(ctx, parsedName, c.registryURL, c.transport); err != nil {
		return &registryRepository{}, err
	}

	return &registryRepository{Repository: repo, client: c}, nil
}

// Search runs a search against the registry, handling dim advanced querying option
func (c *registryClient) Search(query, advanced string, offset, maxResults int) (*t.SearchResults, error) {
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
	values.Set("f", "full")
	values.Set("offset", strconv.Itoa(offset))
	values.Set("maxResults", strconv.Itoa(maxResults))

	httpClient := http.Client{Transport: c.transport}
	if resp, err = httpClient.PostForm(strings.Join([]string{c.registryURL, "/v1/search"}, ""), values); err != nil {
		return nil, fmt.Errorf("Failed to send request : %v", err)
	}
	defer resp.Body.Close()
	var b []byte
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		results := &t.SearchResults{}
		if err := json.NewDecoder(resp.Body).Decode(results); err != nil {
			return nil, fmt.Errorf("Failed to parse response : %v", err)
		}

		return results, nil
	}

	b, _ = ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Server returned an error : %s", string(b))
}

// WalkRepositories walks through all repositories and send them in the given channel
func (c *registryClient) WalkRepositories(repositories chan<- Repository) error {
	return WalkRepositories(c, repositories)
}

// WalkRepositories walks through all repositories and ssend them in the given channel
func WalkRepositories(c Client, repositories chan<- Repository) error {
	var err error

	var n int
	registries := make([]string, 20)
	defer close(repositories)
	last := ""
	for stop := false; !stop; {

		if n, err = c.Repositories(nil, registries, last); err != nil && err != io.EOF {
			logrus.WithField("n", n).WithError(err).Errorln("Failed to get repostories")
			return err
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

			var repository Repository

			if repository, err = c.NewRepository(parsedName); err != nil {
				logrus.WithError(err).WithField("name", last).Errorln("Failed to fetch repository info")
				continue
			}
			repositories <- repository
		}

	}
	return nil

}

// PrintImageInfo prints the info about an image available on the remote registry
func (c *registryClient) PrintImageInfo(w io.Writer, parsedName reference.Named, tpl *template.Template) error {
	var repository Repository
	var err error
	name, _ := reference.ParseNamed(parsedName.Name()[strings.Index(parsedName.Name(), "/")+1:])
	if repository, err = c.NewRepository(name); err != nil {
		logrus.WithError(err).Errorln("Failed to fetch repository info")
		return err
	}

	var tag string
	switch parsedName := parsedName.(type) {
	case reference.NamedTagged:
		tag = parsedName.Tag()
	default:
		tag = "latest"
	}

	var image *Image
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
func (c *registryClient) DeleteImage(parsedName reference.Named) error {
	logrus.WithField("parsedName", parsedName.String()).Debugln("Entering DeleteImage")
	var repo Repository
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
