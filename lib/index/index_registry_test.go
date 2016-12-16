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

package index

import (
	"fmt"
	"io"
	"testing"
	"text/template"

	"sync"

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
	"github.com/nhurel/dim/lib/index/indextest"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/types"
	. "gopkg.in/check.v1"
)

type RegistrySuite struct {
	index *Index
}

type NoOpRegistryClient struct {
	client.Registry
}

func (r *NoOpRegistryClient) Repositories(ctx context.Context, repos []string, last string) (int, error) {
	repos[0] = "httpd"
	repos[1] = "mysql"
	return 2, io.EOF
}

func (r *NoOpRegistryClient) NewRepository(parsedName reference.Named) (registry.Repository, error) {
	return &NoOpRegistryRepository{name: parsedName.Name()}, nil
}
func (r *NoOpRegistryClient) Search(query, advanced string, offset, numResults int) (*types.SearchResults, error) {
	return nil, nil
}

func (r *NoOpRegistryClient) WalkRepositories() <-chan registry.Repository {
	return registry.WalkRepositories(r)
}

func (r *NoOpRegistryClient) PrintImageInfo(w io.Writer, parsedName reference.Named, tpl *template.Template) error {
	return nil
}

func (r *NoOpRegistryClient) DeleteImage(parsedName reference.Named) error {
	return nil
}

func (r *NoOpRegistryClient) ServerVersion() (*types.Info, error) {
	return nil, nil
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
	if digest == "5.7" {
		img = repoImages["mysql:5.7"]
	} else if digest == "3.2" {
		img = repoImages["mongo:3.2"]
	}
	err = nil
	if img == nil {
		err = fmt.Errorf("Image %s not found", digest)
	}
	return
}
func (r *NoOpRegistryRepository) DeleteImage(tag string) error {
	return nil
}

func (r *NoOpRegistryRepository) WalkImages() <-chan *registry.Image {
	return registry.WalkImages(r)
}
func (r *NoOpRegistryRepository) Named() ref.Named {
	n, _ := ref.ParseNamed(r.name)
	return n
}

var repoImages = map[string]*registry.Image{
	"httpd:2.2": {
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "2.2",
		Digest: "httpd:2.2",
	},
	"httpd:2.4": {
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "2.4",
		Digest: "httpd:2.4",
	},
	"mysql:5.5": {
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{},
			},
		},
		Tag:    "5.5",
		Digest: "mysql:5.5",
	},
	"mysql:5.7": {
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{
					Labels: map[string]string{
						"type":   "database",
						"family": "mysql",
					},
					Env: []string{
						"MYSQL_VERSION=5.7",
					},
				},
			},
		},
		Tag:    "5.7",
		Digest: "mysql:5.7",
	},
	"mongo:3.2": {
		Image: &image.Image{
			V1Image: image.V1Image{
				Config: &container.Config{
					Labels: map[string]string{
						"type":   "database",
						"family": "mongodb",
					},
					Env: []string{
						"MONGODB_VERSION=3.2",
					},
				},
			},
		},
		Tag:    "3.2",
		Digest: "mongo:3.2",
	},
}

// Hook up gocheck into the "go test" runner.
func TestRegistry(t *testing.T) { TestingT(t) }

var _ = Suite(&RegistrySuite{})

func (s *RegistrySuite) SetUpTest(c *C) {

	logrus.SetLevel(logrus.DebugLevel)
	var i bleve.Index
	var err error
	if i, err = indextest.MockIndex(ImageMapping); err != nil {
		logrus.WithError(err).Errorln("Failed to create index")
		return
	}
	s.index = &Index{Index: i, RegistryURL: "", RegistryAuth: nil, RegClient: &NoOpRegistryClient{}, Config: &Config{}}
}

func (s *RegistrySuite) TearDownSuite(c *C) {
	s.index.Index.Close()
}

func (s *RegistrySuite) TestBuild(c *C) {
	done := s.index.Build()
	_ = <-done
	srq := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
	srs, err := s.index.Search(srq)
	c.Assert(err, IsNil)
	c.Assert(srs.Total, Equals, uint64(4))
}

func (s *RegistrySuite) TestSearchImages(c *C) {
	done := s.index.Build()
	_ = <-done
	sr, err := s.index.SearchImages("", "+Name:mysql +Tag:5.7", []string{"Name", "Tag", "FullName", "Labels", "Envs"}, 0, 5)
	c.Assert(err, IsNil)
	c.Assert(sr.Total, Equals, uint64(1))
	c.Assert(sr.Hits[0].Fields["Label.family"], Equals, "mysql")
	c.Assert(sr.Hits[0].Fields["Label.type"], Equals, "database")
	c.Assert(sr.Hits[0].Fields["Env.MYSQL_VERSION"], Equals, "5.7")
}

func (s *RegistrySuite) TestGetImageAndIndex(c *C) {
	img, err := s.index.GetImage("mysql", "5.7", digest.FromBytes([]byte("digest")))
	c.Assert(err, IsNil)
	s.index.IndexImage(img)
	sr, err := s.index.SearchImages("", "+Name:mysql +Tag:5.7", []string{"Name", "Tag", "FullName", "Labels", "Envs"}, 0, 5)
	c.Assert(err, IsNil)
	c.Assert(sr.Total, Equals, uint64(1))
	c.Assert(sr.Hits[0].Fields["Label.family"], Equals, "mysql")
	c.Assert(sr.Hits[0].Fields["Label.type"], Equals, "database")
	c.Assert(sr.Hits[0].Fields["Env.MYSQL_VERSION"], Equals, "5.7")
}

func (s *RegistrySuite) TestHandleNotifications(c *C) {
	logrus.Infoln("TestHandleNotifications")
	s.index.Config.Hooks = []*Hook{{Event: PushAction, Action: "{{testCalls}}"}, {Event: DeleteAction, Action: "{{testCalls}}"}}
	s.index.RegClient = &NoOpRegistryClient{}
	s.index.notifications = make(chan *NotificationJob)
	defer close(s.index.notifications)
	calls := make(map[string]int)
	wg := sync.WaitGroup{}
	s.index.Config.RegisterFunction("testCalls", func() error {
		defer wg.Done()
		calls["testCalls"]++
		return nil
	})

	if err := s.index.Config.ParseHooks(); err != nil {
		c.Fatalf("ParseHooks returned an error : %v", err)
	}

	wg.Add(2)

	go func() { s.index.handleNotifications() }()
	s.index.notifications <- &NotificationJob{Action: PushAction, Tag: "3.2", Repository: "mongo"}
	s.index.notifications <- &NotificationJob{Action: DeleteAction, Digest: "mongo:3.2"}
	wg.Wait()

	if calls["testCalls"] != 2 {
		c.Errorf("handleNotifications should have beend called twice but was called %d times", calls["testCalls"])
	}
}
