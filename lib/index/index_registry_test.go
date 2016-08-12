package index

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	ref "github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/types/container"
	"github.com/nhurel/dim/lib/registry"
	. "gopkg.in/check.v1"
	"io"
	"sync"
	"testing"
)

type RegistrySuite struct {
	index *Index
}

type NoOpRegistryClient struct {
	client.Registry
	//registry.RegistryClient
}

func (r *NoOpRegistryClient) Repositories(ctx context.Context, repos []string, last string) (int, error) {
	repos[0] = "httpd"
	repos[1] = "mysql"
	return 2, io.EOF
}

func (r *NoOpRegistryClient) NewRepository(parsedName reference.Named) (registry.Repository, error) {
	return &NoOpRegistryRepository{name: parsedName.Name()}, nil
}
func (r *NoOpRegistryClient) Search(query, advanced string) error {
	return nil
}

func (r *NoOpRegistryClient) WalkRepositories(repositories chan<- registry.Repository) error {
	return registry.WalkRepositories(r, repositories)
}

type NoOpRegistryRepository struct {
	distribution.Repository
	name string
}

func (r *NoOpRegistryRepository) AllTags() ([]string, error) {
	switch r.name {
	case "httpd":
		return []string{"2.2", "2.4"}, nil
	case "mysql":
		return []string{"5.5", "5.7"}, nil
	default:
		return []string{}, nil
	}
}
func (r *NoOpRegistryRepository) Image(tag string) (img *registry.Image, err error) {
	dg := fmt.Sprintf("%s:%s", r.name, tag)
	img = repoImages[dg]
	err = nil
	return
}
func (r *NoOpRegistryRepository) ImageFromManifest(tagDigest digest.Digest, digest string) (img *registry.Image, err error) {
	return nil, nil
}
func (r *NoOpRegistryRepository) DeleteImage(tag string) error {
	return nil
}

func (r *NoOpRegistryRepository) WalkImages(images chan<- *registry.Image) error {
	return registry.WalkImages(r, images)
}
func (r *NoOpRegistryRepository) Named() ref.Named {
	n, _ := ref.ParseNamed(r.name)
	return n
}

var repoImages = map[string]*registry.Image{
	"httpd:2.2": &registry.Image{
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "2.2",
		Digest: "httpd:2.2",
	},
	"httpd:2.4": &registry.Image{
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "2.4",
		Digest: "httpd:2.4",
	},
	"mysql:5.5": &registry.Image{
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "5.5",
		Digest: "mysql:5.5",
	},
	"mysql:5.7": &registry.Image{
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "5.7",
		Digest: "mysql:5.7",
	},
}

// Hook up gocheck into the "go test" runner.
func TestRegistry(t *testing.T) { TestingT(t) }

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) SetUpSuite(c *C) {

	logrus.SetLevel(logrus.DebugLevel)
	fmt.Println("SetupSuite()")
	if i, err := MockIndex(); err != nil {
		logrus.WithError(err).Errorln("Failed to create index")
		return
	} else {
		fmt.Println("New index")
		s.index = &Index{i, "", nil, &NoOpRegistryClient{}, sync.WaitGroup{}}
	}

}

func (s *RegistrySuite) TearDownSuite(c *C) {
	//gock.Off()
}

func (s *RegistrySuite) TestBuild(c *C) {
	s.index.Build()
	fmt.Println("INDEX BUILT")
	srq := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
	srs, err := s.index.Search(srq)
	c.Assert(err, IsNil)
	c.Assert(srs.Total, Equals, uint64(4))
}
