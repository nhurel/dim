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

package dim

import (
	"io"
	"time"

	"text/template"

	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/image"
	"github.com/docker/docker/reference"
)

// SearchResult describes a search result returned from a registry
type SearchResult struct {
	// StarCount indicates the number of stars this repository has (not supported in private registry)
	StarCount int `json:"star_count"`
	// IsOfficial is true if the result is from an official repository. (not supported in private registry)
	IsOfficial bool `json:"is_official"`
	// Name is the name of the repository
	Name string `json:"name"`
	// IsAutomated indicates whether the result is automated (not supported in private registry)
	IsAutomated bool `json:"is_automated"`
	// Description is a textual description of the repository (filled with the tag of the repo)
	Description string `json:"description"`
	// Tag identifie one version of the image
	Tag string `json:"tag"`
	// FullName stores the fully qualified name of the image
	FullName string `json:"full_name"`
	// Created is the time when the image was created
	Created time.Time `json:"created"`
	// Label is an array holding all the labels applied to  an image
	Label map[string]string `json:"label"`
	// Volumes is an array holding all volumes declared by the image
	Volumes []string `json:"volumes"`
	// Exposed port is an array containing all the ports exposed by an image
	ExposedPorts []int `json:"exposed_ports"`
	// Env is a map of all environment variables
	Env map[string]string `json:"env"`
	// Size is the size of the image
	Size int64 `json:"size"`
}

// SearchResults lists a collection search results returned from a registry
type SearchResults struct {
	// Query contains the query string that generated the search results
	Query string `json:"query"`
	// NumResults indicates the number of results the query returned
	NumResults int `json:"num_results"`
	// Results is a slice containing the actual results for the search
	Results []SearchResult `json:"results"`
}

// Info represents the server version endpoint payload
type Info struct {
	// Version returns the version of dim running server side
	Version string `json:"version"`
	// Uptime returns the server uptime in nanoseconds
	Uptime string `json:"uptime"`
}

// RegistryIndex defines method to manage the indexation of a docker registry
type RegistryIndex interface {
	Build() <-chan bool
	GetImage(repository, tag string, dg digest.Digest) (*IndexImage, error)
	IndexImage(image *IndexImage)
	DeleteImage(id string)
	SearchImages(q, a string, fields []string, offset, maxResults int) (*IndexResults, error)
	Submit(job *NotificationJob)
	FindImage(id string) (*IndexImage, error)
}

// RegistryClient defines method to interact with a docker registry
type RegistryClient interface {
	client.Registry
	NewRepository(parsedName reference.Named) (Repository, error)
	Search(query, advanced string, offset, maxResults int) (*SearchResults, error)
	WalkRepositories() <-chan Repository
	PrintImageInfo(out io.Writer, parsedName reference.Named, tpl *template.Template) error
	DeleteImage(parsedName reference.Named) error
	ServerVersion() (*Info, error)
}

// Repository interface defines methods exposed by a registry repository
type Repository interface {
	distribution.Repository
	AllTags() ([]string, error)
	Image(tag string) (img *RegistryImage, err error)
	ImageFromManifest(tagDigest digest.Digest, tag string) (img *RegistryImage, err error)
	DeleteImage(tag string) error
	WalkImages() <-chan *RegistryImage
}

// RegistryImage is an Image representation from the registry
type RegistryImage struct {
	*image.Image
	Tag    string
	Digest string
}

// IndexImage is an Image modeling for indexation
type IndexImage struct {
	ID           string
	Name         string
	FullName     string
	Tag          string
	Comment      string
	Created      time.Time
	Author       string
	Label        map[string]string
	Labels       []string
	Volumes      []string
	ExposedPorts []int
	Env          map[string]string
	Envs         []string
	Size         int64
}

// Type implementation of bleve.Classifier interface
func (im IndexImage) Type() string {
	return "image"
}

// IndexResults contains all results of a given index search
type IndexResults struct {
	Total  uint64
	Images []*IndexImage
}

// ActionType indicates the kind of a NotificationJob
type ActionType string

// DeleteAction indicates a NotificationJob should delete an image from the index
const DeleteAction ActionType = "delete"

// PushAction indicates a NotificationJob should add or update an image in the index
const PushAction ActionType = "push"

// NotificationJob stores info to reindex an image after a push or deletion
type NotificationJob struct {
	Action     ActionType
	Repository string
	Tag        string
	Digest     digest.Digest
}

// RegistryProxy forwards request to a docker registry if user is granted
type RegistryProxy interface {
	Forwards(w http.ResponseWriter, r *http.Request)
}
