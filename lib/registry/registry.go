package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/registry"
	"github.com/docker/engine-api/types"
	reg "github.com/docker/engine-api/types/registry"
	"github.com/howeyc/gopass"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

type Client interface {
	client.Registry
	NewRepository(parsedName reference.Named) (Repository, error)
	Search(query, advanced string) error
	WalkRepositories(repositories chan<- Repository) error
}

type RegistryClient struct {
	client.Registry
	transport   http.RoundTripper
	registryUrl string
}

var ctx = context.Background()

// Create a registry client. Handles getting right credentials from user
func New(registryAuth *types.AuthConfig, registryUrl string) (Client, error) {
	var err error
	var reg client.Registry

	transport := http.DefaultTransport

	if registryAuth != nil {
		transport = registry.AuthTransport(transport, registryAuth, true)
	}

	if reg, err = client.NewRegistry(ctx, registryUrl, transport); err != nil {
		return nil, err
	}

	repos := make([]string, 1)
	for _, err = reg.Repositories(ctx, repos, ""); err != nil && err != io.EOF; _, err = reg.Repositories(ctx, repos, "") {
		logrus.Debugln("Prompting for credentials")
		if registryAuth == nil {
			registryAuth = &types.AuthConfig{}
		}
		if registryAuth.Username != "" {
			fmt.Printf("Username (%s) :", registryAuth.Username)
		} else {
			fmt.Print("Username :")
		}
		var input string
		fmt.Scanln(&input)
		if input != "" {
			registryAuth.Username = input
		} else if registryAuth.Username == "" {
			return nil, err
		}
		fmt.Print("Password :")
		pwd, _ := gopass.GetPasswd()
		input = string(pwd)
		if input == "" {
			return nil, err
		}
		registryAuth.Password = input
		transport = registry.AuthTransport(transport, registryAuth, true)
		if reg, err = client.NewRegistry(ctx, registryUrl, transport); err != nil {
			return nil, err
		}
	}

	return &RegistryClient{reg, transport, registryUrl}, nil
}

// Create a Repository object to query the registry about a specific repository
func (c *RegistryClient) NewRepository(parsedName reference.Named) (Repository, error) {
	logrus.WithField("name", parsedName).Debugln("Creating new repository")
	if repo, err := client.NewRepository(ctx, parsedName, c.registryUrl, c.transport); err != nil {
		return &RegistryRepository{}, err
	} else {
		return &RegistryRepository{Repository: repo, client: c}, nil
	}
}

// Runs a search against the registry, handling dim advanced querying option
func (c *RegistryClient) Search(query, advanced string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	q := strings.TrimSpace(query)
	a := strings.TrimSpace(advanced)
	var err error
	if q != "" {
		if err = writer.WriteField("q", q); err != nil {
			return err
		}
	}
	if a != "" {
		if err = writer.WriteField("a", a); err != nil {
			return err
		}
	}
	if err = writer.Close(); err != nil {
		return err
	}

	var req *http.Request

	if req, err = http.NewRequest(http.MethodPost, strings.Join([]string{c.registryUrl, "/v1/search"}, ""), body); err != nil {
		return fmt.Errorf("Failed to create request : %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var resp *http.Response

	logrus.WithField("request", req.Body).Debugln("Sending request")
	//FIXME  : Use http.PostForm("url", url.Values{"q": query, "a":advanced}) instead

	httpClient := http.Client{Transport: c.transport}
	if resp, err = httpClient.Do(req); err != nil {
		return fmt.Errorf("Failed to send request : %v", err)
	}
	defer resp.Body.Close()
	var b []byte
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		results := &reg.SearchResults{}
		if err := json.NewDecoder(resp.Body).Decode(results); err != nil {
			return fmt.Errorf("Failed to parse response : %v", err)
		}

		t, _ := template.New("search").Parse(searchResultTemplate)
		w := tabwriter.NewWriter(os.Stdout, 8, 8, 8, ' ', 0)
		if err = t.Execute(w, results); err != nil {
			logrus.WithError(err).Errorln("Failed to parse template")
		}
		w.Flush()

	} else {
		b, _ = ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Server returned an error", string(b))
	}
	return nil

}

const searchResultTemplate = `
{{.NumResults}} Results found :
Name	Tag	Automated	Official
{{ range $i, $r := .Results}} {{- $r.Name}}	{{$r.Description}}	{{$r.IsAutomated}}	{{$r.IsOfficial }}
{{end}}
`

func (c *RegistryClient) WalkRepositories(repositories chan<- Repository) error {
	return WalkRepositories(c, repositories)
}

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
