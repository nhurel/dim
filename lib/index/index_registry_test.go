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
	"testing"

	"fmt"
	"io"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/image"
	dockerReference "github.com/docker/docker/reference"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/index/indextest"
	"github.com/nhurel/dim/lib/mock"
	"github.com/nhurel/dim/lib/registry"
	. "gopkg.in/check.v1"
)

type RegistrySuite struct {
	index              *Index
	mockRegistryClient *mock.NoOpRegistryClient
}

var repoImages = map[string]*dim.RegistryImage{
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

	mockRegistryClient := &mock.NoOpRegistryClient{}
	mockRegistryClient.RepositoriesFn = func(repos []string, last string) (int, error) {
		repos[0] = "httpd"
		repos[1] = "mysql"
		return 2, io.EOF
	}
	mockRegistryClient.WalkRepoitoriesFn = func() <-chan dim.Repository {
		return registry.WalkRepositories(mockRegistryClient)
	}

	mockRegistryRepository := map[string]*mock.NoOpRegistryRepository{
		"httpd": {
			ImageFn: func(tag string) (*dim.RegistryImage, error) {
				dg := fmt.Sprintf("httpd:%s", tag)
				return repoImages[dg], nil
			},
			ImageFromManifestFn: func(tagDigest digest.Digest, digest string) (img *dim.RegistryImage, err error) {
				if digest == "5.7" {
					img = repoImages["mysql:5.7"]
				}
				err = nil
				if img == nil {
					err = fmt.Errorf("Image %s not found", digest)
				}
				return
			},
			NamedFn: func() reference.Named {
				n, _ := reference.ParseNamed("httpd")
				return n
			},
			AllTagsFn: func() ([]string, error) {
				return []string{"2.2", "2.4"}, nil
			},
		},
		"mysql": {
			ImageFn: func(tag string) (*dim.RegistryImage, error) {
				dg := fmt.Sprintf("mysql:%s", tag)
				return repoImages[dg], nil
			},
			ImageFromManifestFn: func(tagDigest digest.Digest, digest string) (img *dim.RegistryImage, err error) {
				if digest == "5.7" {
					img = repoImages["mysql:5.7"]
				}
				err = nil
				if img == nil {
					err = fmt.Errorf("Image %s not found", digest)
				}
				return
			},
			NamedFn: func() reference.Named {
				n, _ := reference.ParseNamed("mysql")
				return n
			},
			AllTagsFn: func() ([]string, error) {
				return []string{"5.5", "5.7"}, nil
			},
		},
	}

	mockRegistryRepository["httpd"].WalkImagesFn = func() <-chan *dim.RegistryImage { return registry.WalkImages(mockRegistryRepository["httpd"]) }
	mockRegistryRepository["mysql"].WalkImagesFn = func() <-chan *dim.RegistryImage { return registry.WalkImages(mockRegistryRepository["mysql"]) }

	mockRegistryClient.NewRepositoryFn = func(parsedName dockerReference.Named) (dim.Repository, error) {
		return mockRegistryRepository[parsedName.Name()], nil
	}

	s.index = &Index{Index: i, RegClient: mockRegistryClient, Config: &Config{}}
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
	c.Assert(sr.Images[0].Label["family"], Equals, "mysql")
	c.Assert(sr.Images[0].Label["type"], Equals, "database")
	c.Assert(sr.Images[0].Env["MYSQL_VERSION"], Equals, "5.7")
}

func (s *RegistrySuite) TestGetImageAndIndex(c *C) {
	img, err := s.index.GetImage("mysql", "5.7", digest.FromBytes([]byte("digest")))
	c.Assert(err, IsNil)
	s.index.IndexImage(img)
	sr, err := s.index.SearchImages("", "+Name:mysql +Tag:5.7", []string{"Name", "Tag", "FullName", "Labels", "Envs"}, 0, 5)
	c.Assert(err, IsNil)
	c.Assert(sr.Total, Equals, uint64(1))
	c.Assert(sr.Images[0].Label["family"], Equals, "mysql")
	c.Assert(sr.Images[0].Label["type"], Equals, "database")
	c.Assert(sr.Images[0].Env["MYSQL_VERSION"], Equals, "5.7")
}

func (s *RegistrySuite) TestHandleNotifications(c *C) {
	logrus.Infoln("TestHandleNotifications")
	s.index.Config.Hooks = []*Hook{{Event: dim.PushAction, Action: "{{testCalls}}"}, {Event: dim.DeleteAction, Action: "{{testCalls}}"}}
	s.index.RegClient = &mock.NoOpRegistryClient{
		NewRepositoryFn: func(parsedName dockerReference.Named) (dim.Repository, error) {
			return &mock.NoOpRegistryRepository{
				ImageFn: func(tag string) (*dim.RegistryImage, error) {
					dg := fmt.Sprintf("mongo:%s", tag)
					return repoImages[dg], nil
				},
				ImageFromManifestFn: func(tagDigest digest.Digest, digest string) (img *dim.RegistryImage, err error) {
					return repoImages["mongo:3.2"], nil
				},
				NamedFn: func() reference.Named {
					n, _ := reference.ParseNamed("mongo")
					return n
				},
				AllTagsFn: func() ([]string, error) {
					return []string{"3.2"}, nil
				},
			}, nil
		},
	}
	s.index.notifications = make(chan *dim.NotificationJob)
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
	s.index.notifications <- &dim.NotificationJob{Action: dim.PushAction, Tag: "3.2", Repository: "mongo"}
	s.index.notifications <- &dim.NotificationJob{Action: dim.DeleteAction, Digest: "mongo:3.2"}
	wg.Wait()

	if calls["testCalls"] != 2 {
		c.Errorf("handleNotifications should have beend called twice but was called %d times", calls["testCalls"])
	}
}
