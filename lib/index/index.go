package index

import (
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib/registry"
	"strings"
	"sync"
)

type Index struct {
	bleve.Index
	registryUrl  string
	registryAuth *types.AuthConfig
	regClient    registry.Client
	buildWg      sync.WaitGroup
}

// New create a new instance to manage a index of a given registry into a specific directory
func New(dir string, registryUrl string, registryAuth *types.AuthConfig) (*Index, error) {
	var i bleve.Index
	var reg registry.Client
	var err error

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", imageMapping)
	if i, err = bleve.New(dir, mapping); err != nil {
		return nil, err
	}

	if reg, err = registry.New(registryAuth, registryUrl); err != nil {
		return nil, err
	}

	return &Index{i, registryUrl, registryAuth, reg, sync.WaitGroup{}}, nil
}

// Build creates a full index from the registry
func (idx *Index) Build() {

	repositories := make(chan registry.Repository, 10)
	go idx.regClient.WalkRepositories(repositories)

	for repository := range repositories {
		idx.buildWg.Add(1)
		go func(repo registry.Repository) {
			defer idx.buildWg.Done()
			if err := idx.indexRepository(repo); err != nil {
				logrus.WithError(err).WithField("repository", repo.Named().Name()).Errorln("An error occured while indexin repository")
			}
		}(repository)
	}
	idx.buildWg.Wait()
}

// indexRepository browse all tags of a given repository and index the corresponding images
func (idx *Index) indexRepository(repository registry.Repository) error {
	l := logrus.WithField("repository", repository.Named().Name())

	l.Infoln("Indexing repository")

	images := make(chan *registry.Image, 10)
	go repository.WalkImages(images)

	for img := range images {
		idx.buildWg.Add(1)
		go func(n string, i *registry.Image) {
			defer idx.buildWg.Done()
			idx.IndexImage(Parse(n, i))
		}(repository.Named().Name(), img)
	}

	return nil
}

func (idx *Index) GetImageAndIndex(repository, tag string, dg digest.Digest) error {
	named, _ := reference.ParseNamed(repository)
	if repo, err := idx.regClient.NewRepository(named); err != nil {
		logrus.WithError(err).WithField("Repository", repository).Errorln("Failed get repository info")
		return err
	} else {
		if img, err := repo.ImageFromManifest(dg, tag); err != nil {
			logrus.WithError(err).Errorln("Failed to get image info from manifest")
			return err
		} else {
			idx.IndexImage(Parse(repository, img))
		}
	}
	return nil
}

// IndexImage adds a given image into the index
func (idx *Index) IndexImage(image *Image) {
	logrus.WithFields(logrus.Fields{"imageID": image.ID, "image.FullName": image.FullName}).Debugln("Indexing image")
	idx.Index.Index(image.FullName, image)
}

// IndexImage adds a given image into the index
func (idx *Index) DeleteImage(id string) {
	l := logrus.WithField("imageID", id)
	l.Debugln("Removing image from index")
	q := bleve.NewTermQuery(id).SetField("ID")
	rq := bleve.NewSearchRequest(q)
	rq.Fields = []string{"FullName"}
	var sr *bleve.SearchResult
	var err error
	if sr, err = idx.Search(rq); err != nil || sr.Total == 0 {
		l.WithError(err).WithField("#hits", sr.Total).Errorln("Failed to get image id to remove from index")
		return
	}
	if sr.Total > 1 {
		l.WithField("#hits", sr.Total).Warnln("Removing multiple images from index for this imageID")
		return
	}

	for _, h := range sr.Hits {
		l.WithField("image.FullName", h.Fields["FullName"].(string)).Infoln("Removing image from index")
		idx.Index.Delete(h.Fields["FullName"].(string))
	}
}

func (idx *Index) BuildQuery(nameTag, advanced string) bleve.Query {
	l := logrus.WithFields(logrus.Fields{"nameTag": nameTag, "advanced": advanced})
	l.Debugln("Building query clause")
	bq := make([]bleve.Query, 0, 3)

	name := nameTag
	tag := nameTag

	if split := strings.Split(nameTag, ":"); len(split) == 2 {
		name = split[0]
		tag = split[1]
	}

	if nameTag != "" {
		l.WithFields(logrus.Fields{"name": name, "tag": tag}).Debugln("Adding name and tag clauses")
		bq = append(bq, bleve.NewFuzzyQuery(name).SetField("Name"), bleve.NewMatchQuery(tag).SetField("Tag"))
	}

	if advanced != "" {
		l.Debugln("Adding advanced clause")
		bq = append(bq, bleve.NewQueryStringQuery(advanced))
	}

	logrus.WithField("queries", bq).Debugln("Returning query with should clauses")
	return bleve.NewBooleanQuery(nil, bq, nil)

}
