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

package mock

import (
	"io"
	"text/template"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	ref "github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/reference"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/nhurel/dim/lib"
)

// NoOpRegistryClient is a mock implementation of dim.RegistryClient
type NoOpRegistryClient struct {
	client.Registry
	RepositoriesFn    func(repos []string, last string) (int, error)
	WalkRepoitoriesFn func() <-chan dim.Repository
	NewRepositoryFn   func(parsedName reference.Named) (dim.Repository, error)
}

// Repositories is a mock implementation of Repositories method of dim.RegistryClient interface
func (r *NoOpRegistryClient) Repositories(ctx context.Context, repos []string, last string) (int, error) {
	return r.RepositoriesFn(repos, last)
}

// NewRepository is a mock implementation of NewRepository method of dim.RegistryClient interface
func (r *NoOpRegistryClient) NewRepository(parsedName reference.Named) (dim.Repository, error) {
	return r.NewRepositoryFn(parsedName)
}

// Search is a mock implementation of Search method of dim.RegistryClient interface
func (r *NoOpRegistryClient) Search(query, advanced string, offset, numResults int) (*dim.SearchResults, error) {
	return nil, nil
}

// WalkRepositories is a mock implementation of WalkRepositories method of dim.RegistryClient interface
func (r *NoOpRegistryClient) WalkRepositories() <-chan dim.Repository {
	return r.WalkRepoitoriesFn()
}

// PrintImageInfo is a mock implementation of PrintImageInfo method of dim.RegistryClient interface
func (r *NoOpRegistryClient) PrintImageInfo(w io.Writer, parsedName reference.Named, tpl *template.Template) error {
	return nil
}

// DeleteImage is a mock implementation of DeleteImage method of dim.RegistryClient interface
func (r *NoOpRegistryClient) DeleteImage(parsedName reference.Named) error {
	return nil
}

// ServerVersion is a mock implementation of ServerVersion method of dim.RegistryClient interface
func (r *NoOpRegistryClient) ServerVersion() (*dim.Info, error) {
	return nil, nil
}

// NoOpRegistryRepository is a mock implementation of dim.Repository interface
type NoOpRegistryRepository struct {
	distribution.Repository
	Name                string
	AllTagsFn           func() ([]string, error)
	ImageFn             func(tag string) (*dim.RegistryImage, error)
	ImageFromManifestFn func(tagDigest digest.Digest, digest string) (img *dim.RegistryImage, err error)
	WalkImagesFn        func() <-chan *dim.RegistryImage
	NamedFn             func() ref.Named
}

// AllTags is a mock implementation of AllTags method from dim.Repository interface
func (r *NoOpRegistryRepository) AllTags() ([]string, error) {
	return r.AllTagsFn()
}

// Image is a mock implementation of Image method from dim.Repository interface
func (r *NoOpRegistryRepository) Image(tag string) (*dim.RegistryImage, error) {
	return r.ImageFn(tag)
}

// ImageFromManifest is a mock implementation of ImageFromManifest method from dim.Repository interface
func (r *NoOpRegistryRepository) ImageFromManifest(tagDigest digest.Digest, digest string) (*dim.RegistryImage, error) {
	return r.ImageFromManifestFn(tagDigest, digest)
}

// DeleteImage is a mock implementation of DeleteImage method from dim.Repository interface
func (r *NoOpRegistryRepository) DeleteImage(tag string) error {
	return nil
}

// WalkImages is a mock implementation of WalkImages method from dim.Repository interface
func (r *NoOpRegistryRepository) WalkImages() <-chan *dim.RegistryImage {
	return r.WalkImagesFn()
}

// Named is a mock implementation of Named method from dim.Repository interface
func (r *NoOpRegistryRepository) Named() ref.Named {
	return r.NamedFn()
}

// NoOpRegistryIndex is a mock implementation of dim.RegistryIndex interface
type NoOpRegistryIndex struct {
	Calls map[string][]interface{}
}

// Build is a mock implementation of Build method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) Build() <-chan bool {
	i.Calls["Build"] = nil
	return nil
}

// GetImage is a mock implementation of GetImage method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) GetImage(repository, tag string, dg digest.Digest) (*dim.IndexImage, error) {
	i.Calls["GetImageAndIndex"] = []interface{}{repository, tag, dg}
	return nil, nil
}

// IndexImage is a mock implementation of IndexImage method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) IndexImage(image *dim.IndexImage) {
	i.Calls["IndexImage"] = []interface{}{image}
}

// DeleteImage is a mock implementation of DeleteImage method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) DeleteImage(id string) {
	i.Calls["DeleteImage"] = []interface{}{id}
}

// SearchImages is a mock implementation of SearchImages method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) SearchImages(q, a string, fields []string, offset, maxResults int) (*dim.IndexResults, error) {
	i.Calls["SearchImages"] = []interface{}{q, a, fields, offset, maxResults}
	return nil, nil
}

// Submit is a mock implementation of Submit method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) Submit(job *dim.NotificationJob) {
	i.Calls["Submit"] = []interface{}{job}
}

// FindImage is a mock implementation of FindImage method from dim.RegistryIndex interface
func (i *NoOpRegistryIndex) FindImage(id string) (*dim.IndexImage, error) {
	i.Calls["FindImage"] = []interface{}{id}
	return nil, nil
}

// NoOpDockerClient is a mock implementation of dockerClient.Docker interface
type NoOpDockerClient struct {
	ImageInspectLabels map[string]string
	Calls              map[string][]interface{}
}

// ImageBuild is a mock implementation of ImageBuild method from dockerClient.Client interface
func (n *NoOpDockerClient) ImageBuild(parent string, buildLabels map[string]string, tag string) error {
	n.Calls["ImageBuild"] = []interface{}{parent, buildLabels, tag}
	return nil
}

// Pull is a mock implementation of Pull method from dockerClient.Client interface
func (n *NoOpDockerClient) Pull(image string) error {
	n.Calls["Pull"] = []interface{}{image}
	return nil
}

// Inspect is a mock implementation of Inspect method from dockerClient.Client interface
func (n *NoOpDockerClient) Inspect(image string) (types.ImageInspect, error) {
	n.Calls["Inspect"] = []interface{}{image}
	return types.ImageInspect{Config: &container.Config{Labels: n.ImageInspectLabels}, ContainerConfig: &container.Config{Labels: n.ImageInspectLabels}}, nil
}

// Remove is a mock implementation of Remove method from dockerClient.Client interface
func (n *NoOpDockerClient) Remove(image string) error {
	n.Calls["Remove"] = []interface{}{image}
	return nil
}

// Push is a mock implementation of Push method from dockerClient.Client interface
func (n *NoOpDockerClient) Push(image string) error {
	n.Calls["Push"] = []interface{}{image}
	return nil
}
