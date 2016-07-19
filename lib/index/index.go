package index

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/distribution"
	_ "github.com/docker/distribution/manifest/schema1"
	_ "github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
	"github.com/docker/docker/registry"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"io"
	"net/http"
)

type Index struct {
	bleve.Index
	registryUrl  string
	registryAuth *types.AuthConfig
}

// New create a new instance to manage a index of a given registry into a specific directory
func New(dir string, registryUrl string, registryAuth *types.AuthConfig) (*Index, error) {
	var i bleve.Index
	var err error

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", imageMapping)
	if i, err = bleve.New(dir, mapping); err != nil {
		return nil, err
	}
	return &Index{i, registryUrl, registryAuth}, nil
}

// Build creates a full index from the registry
func (idx *Index) Build() error {
	var reg client.Registry
	var err error

	background := context.Background()

	transport := http.DefaultTransport
	if idx.registryAuth != nil {
		transport = registry.AuthTransport(transport, idx.registryAuth, true)
	}

	if reg, err = client.NewRegistry(background, idx.registryUrl, transport); err != nil {
		return err
	}

	var n int
	registries := make([]string, 20)
	last := ""

	// TODO : Loop until there are no more repositories
	if n, err = reg.Repositories(background, registries, last); err != io.EOF {
		logrus.WithField("n", n).WithError(err).Errorln("Failed to get repostories")
		return err
	}

	var parsedName reference.Named
	for i := 0; i < n; i++ {
		last = registries[i]
		l := logrus.WithField("repository", last)
		l.Debugln("Indexing repository")
		if parsedName, err = reference.ParseNamed(last); err != nil {
			logrus.WithError(err).WithField("name", last).Errorln("Failed to parse repository name")
			return err
		}

		var repository distribution.Repository

		if repository, err = client.NewRepository(background, parsedName, idx.registryUrl, transport); err != nil {
			logrus.WithError(err).WithField("name", last).Errorln("Failed to fetch repository info")
			return err
		}

		var tags []string

		tService := repository.Tags(background)
		if tags, err = tService.All(background); err != nil {
			logrus.WithField("repository", repository.Named().Name()).WithError(err).Errorln("Failed to get tags ")
			return err
		}
		var mService distribution.ManifestService

		if mService, err = repository.Manifests(background); err != nil {
			logrus.WithError(err).Errorln("Failed to instantiate manifestService")
			return err
		}

		for _, tag := range tags {
			l = l.WithField("tag", tag)
			l.Debugln("Getting tag details")
			var tDescriptor distribution.Descriptor
			if tDescriptor, err = tService.Get(background, tag); err != nil {
				logrus.WithFields(logrus.Fields{"repository": repository.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to get Tag")
				return err
			}

			var mf distribution.Manifest
			l = l.WithField("tagDigest", tDescriptor.Digest)
			l.Debugln("Getting manifest")
			if mf, err = mService.Get(background, tDescriptor.Digest, distribution.WithTag(tag)); err != nil {
				logrus.WithFields(logrus.Fields{"repository": repository.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to get manifest")
				return err
			}

			l.Debugln("Reading manifest")
			var payload []byte
			if _, payload, err = mf.Payload(); err != nil {
				logrus.WithError(err).Errorln("Failed to read manifest")
				return err
			}

			l.Debugln("Unmarshalling manifest")
			im := &ImageManifest{}
			if err = json.Unmarshal(payload, im); err != nil {
				logrus.WithFields(logrus.Fields{"repository": repository.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to read image manifest")
				return err
			}

			img := &image.V1Image{}
			l = l.WithField("imageManifest", im)
			l.Debugln("Reading image")
			if err = json.Unmarshal([]byte(im.History[0]["v1Compatibility"]), img); err != nil {
				logrus.WithFields(logrus.Fields{"repository": repository.Named().Name(), "tag": tag}).WithError(err).Errorln("Failed to unmarshall image info")
				return err
			}

			l.WithField("image", img).Debugln("Indexing image")

			go func(n, t string, i *image.V1Image) {
				idx.IndexImage(Parse(n, t, i))
			}(im.Name, im.Tag, img)
		}
	}

	return nil
}

// IndexImage adds a given image into the index
func (idx *Index) IndexImage(image *Image) {
	idx.Index.Index(image.ID, image)
}

type ImageManifest struct {
	Name    string              `json:"name,omitempty"`
	Tag     string              `json:"tag,omitempty"`
	History []map[string]string `json:"history,omitempty"`
}
