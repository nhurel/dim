package index

import (
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/registry"
	"golang.org/x/net/context"
	"io"
)

type Index struct {
	bleve.Index
	registryUrl  string
	registryAuth *types.AuthConfig
	regClient    *registry.Client
}

// New create a new instance to manage a index of a given registry into a specific directory
func New(dir string, registryUrl string, registryAuth *types.AuthConfig) (*Index, error) {
	var i bleve.Index
	var reg *registry.Client
	var err error

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", imageMapping)
	if i, err = bleve.New(dir, mapping); err != nil {
		return nil, err
	}

	if reg, err = registry.New(registryAuth, registryUrl); err != nil {
		return nil, err
	}

	return &Index{i, registryUrl, registryAuth, reg}, nil
}

// Build creates a full index from the registry
func (idx *Index) Build() error {
	var err error

	var n int
	registries := make([]string, 20)
	last := ""
	for stop := false; !stop; {

		if n, err = idx.regClient.Repositories(nil, registries, last); err != nil && err != io.EOF {
			logrus.WithField("n", n).WithError(err).Errorln("Failed to get repostories")
			return err
		}
		stop = (err == io.EOF)

		for i := 0; i < n; i++ {
			last = registries[i]

			if err = idx.indexRepository(last, nil); err != nil {
				return err
			}
		}

	}

	return nil
}

// indexRepository browse all tags of a given repository and index the corresponding images
func (idx *Index) indexRepository(repo string, ctx context.Context) error {
	var parsedName reference.Named
	var err error

	l := logrus.WithField("repository", repo)
	l.Infoln("Indexing repository")
	if parsedName, err = reference.ParseNamed(repo); err != nil {
		logrus.WithError(err).WithField("name", repo).Errorln("Failed to parse repository name")
		return err
	}

	var repository registry.Repository

	if repository, err = idx.regClient.NewRepository(parsedName); err != nil {
		logrus.WithError(err).WithField("name", repo).Errorln("Failed to fetch repository info")
		return err
	}

	var tags []string

	if tags, err = repository.AllTags(); err != nil {
		logrus.WithField("repository", repository.Named().Name()).WithError(err).Errorln("Failed to get tags ")
		return err
	}

	for _, tag := range tags {
		l = l.WithField("tag", tag)
		l.Debugln("Getting image details")

		var img *image.Image
		var id string
		if id, img, err = repository.Image(tag); err != nil {
			return err
		}

		l.WithField("image", img).Debugln("Indexing image")

		go func(d, n, t string, i *image.Image) {
			idx.IndexImage(Parse(d, n, t, i))
		}(id, repo, tag, img)
	}

	return nil
}

// IndexImage adds a given image into the index
func (idx *Index) IndexImage(image *Image) {
	idx.Index.Index(image.ID, image)
}

func (idx *Index) BuildQuery(nameTag, advanced string) bleve.Query {
	bq := make([]bleve.Query, 0, 3)

	if nameTag != "" {
		bq = append(bq, bleve.NewFuzzyQuery(nameTag).SetField("Name"), bleve.NewFuzzyQuery(nameTag).SetField("Tag"))
	}

	if advanced != "" {
		bq = append(bq, bleve.NewQueryStringQuery(advanced))
	}

	logrus.WithField("queries", bq).Debugln("Returning query with should clauses")
	return bleve.NewBooleanQuery(nil, bq, nil)

}
