package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/registry"
	"github.com/docker/engine-api/types"
	reg "github.com/docker/engine-api/types/registry"
	"github.com/howeyc/gopass"
	"golang.org/x/net/context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

type Client struct {
	client.Registry
	transport   http.RoundTripper
	registryUrl string
}

type Repository struct {
	distribution.Repository
	client *Client
	mfs    distribution.ManifestService
	bfs    distribution.BlobService
}

var ctx = context.Background()

// Create a registry client. Handles getting right credentials from user
func New(registryAuth *types.AuthConfig, registryUrl string) (*Client, error) {
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
	for _, err = reg.Repositories(ctx, repos, ""); err != nil; _, err = reg.Repositories(ctx, repos, "") {
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

	return &Client{reg, transport, registryUrl}, nil
}

// Create a Repository object to query the registry about a specific repository
func (c *Client) NewRepository(parsedName reference.Named) (Repository, error) {
	logrus.WithField("name", parsedName).Debugln("Creating new repository")
	if repo, err := client.NewRepository(ctx, parsedName, c.registryUrl, c.transport); err != nil {
		return Repository{}, err
	} else {
		return Repository{Repository: repo, client: c}, nil
	}
}

// Runs a search against the registry, handling dim advanced querying option
func (c *Client) Search(query, advanced string) error {
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

// AllTags returns all existing tag for this repository instance
func (r *Repository) AllTags() ([]string, error) {
	return r.tagService().All(ctx)
}

func (r *Repository) tagService() distribution.TagService {
	return r.Repository.Tags(ctx)
}

func (r *Repository) manifestService() (distribution.ManifestService, error) {
	if r.mfs == nil {
		if mService, err := r.Manifests(ctx); err != nil {
			logrus.WithError(err).Errorln("Failed to instantiate manifestService")
			return nil, err
		} else {
			r.mfs = mService
		}
	}
	return r.mfs, nil
}

func (r *Repository) blobService() distribution.BlobService {
	if r.bfs == nil {
		r.bfs = r.Blobs(ctx)
	}
	return r.bfs
}

// Image return image info for a given tag
func (r *Repository) Image(tag string) (dg string, img *image.Image, err error) {

	var tagDigest digest.Digest
	if tagDigest, err = r.getTagDigest(tag); err != nil {
		return
	}

	dg = string(tagDigest)
	if img, err = r.ImageFromManifest(tagDigest); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to get image")
	}
	return
}

// ImageFromManifest returns image inforamtion from its manifest digest
func (r *Repository) ImageFromManifest(tagDigest digest.Digest) (img *image.Image, err error) {
	var mService distribution.ManifestService
	if mService, err = r.manifestService(); err != nil {
		return
	}

	var mf distribution.Manifest
	l := logrus.WithField("tagDigest", tagDigest)
	l.Debugln("Getting manifest")
	if mf, err = mService.Get(ctx, tagDigest); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to get manifest")
		return
	}

	l.Debugln("Reading manifest")
	var payload []byte
	if _, payload, err = mf.Payload(); err != nil {
		logrus.WithError(err).Errorln("Failed to read manifest")
		return
	}

	l.Debugln("Unmarshalling manifest")
	manif := &schema2.Manifest{}
	if err = json.Unmarshal(payload, manif); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name()}).WithError(err).Errorln("Failed to read image manifest")
		return
	}

	if payload, err = r.blobService().Get(ctx, manif.Config.Digest); err != nil {
		logrus.WithError(err).Errorln("Failed to get image config")
		return
	}

	logrus.WithField("Digest", manif.Config.Digest).Debugln("Unmarshalling V2Image")

	img = &image.Image{}
	if err = json.Unmarshal(payload, img); err != nil {
		logrus.WithField("Digest", manif.Config.Digest).WithError(err).Errorln("Failed to read image")
		return
	}

	return
}

func (r *Repository) getTagDigest(tag string) (digest.Digest, error) {
	var err error
	var tDescriptor distribution.Descriptor
	if tDescriptor, err = r.tagService().Get(ctx, tag); err != nil {
		logrus.WithFields(logrus.Fields{"repository": r.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to get Tag")
		return "", err
	}
	return tDescriptor.Digest, nil

}

// DeleteImage sends a delete request on tag
func (r *Repository) DeleteImage(tag string) error {
	logrus.WithField("tag", tag).Debugln("Entering DeleteImage")
	var err error
	var mfService distribution.ManifestService

	var tagDigest digest.Digest
	if tagDigest, err = r.getTagDigest(tag); err != nil {
		return err
	}

	if mfService, err = r.manifestService(); err != nil {
		return err
	}

	logrus.WithField("tagDigest", tagDigest).Debugln("Calling delete on manifestService")
	return mfService.Delete(ctx, tagDigest)
}

type ImageManifest struct {
	Name    string              `json:"name,omitempty"`
	Tag     string              `json:"tag,omitempty"`
	History []map[string]string `json:"history,omitempty"`
}
