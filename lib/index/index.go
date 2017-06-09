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
	"strings"
	"sync"

	"time"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/reference"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/utils"
)

// Index manages indexation of docker images
type Index struct {
	// Index is the bleve.Index instance
	bleve.Index
	Config        *Config
	RegClient     dim.RegistryClient
	notifications chan *dim.NotificationJob
}

type repoImage struct {
	repoName string
	image    *dim.RegistryImage
}

// New create a new instance to manage a index of a given registry into a specific directory
func New(cfg *Config, regClient dim.RegistryClient) (*Index, error) {
	var i bleve.Index
	var err error

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", ImageMapping)
	if i, err = bleve.New(cfg.Directory, mapping); err != nil {
		return nil, err
	}

	notifications := make(chan *dim.NotificationJob, 3)
	index := &Index{Index: i, RegClient: regClient, notifications: notifications, Config: cfg}
	index.loop(3)
	return index, nil
}

// Build creates a full index from the registry.
// The returned channel is closed once all images are indexed so the caller can block until the index is built if needed
func (idx *Index) Build() <-chan bool {
	// Channel to indicate to the caller when the indexation is done
	done := make(chan bool, 1)

	go func() {

		repositories := idx.RegClient.WalkRepositories()

		// Channel to browse all repository images
		images := make(chan *repoImage, 10)

		// Waitgoup to watch when all repo images have been read and pushed to images channel
		browseImgWg := sync.WaitGroup{}
		for repository := range repositories {
			browseImgWg.Add(1)
			go func(repo dim.Repository) {
				defer browseImgWg.Done()
				for img := range repo.WalkImages() {
					images <- &repoImage{repo.Named().Name(), img}
				}
			}(repository)
		}

		// Channel to push parsed images, erady o be indexed
		tasks := make(chan *dim.IndexImage, 5)

		// Waitgroup to watch when all image have been parsed and pushed i tasks channel
		parseImgWg := sync.WaitGroup{}
		parseImgWg.Add(3)
		for i := 0; i < 3; i++ {
			go func() {
				defer parseImgWg.Done()
				for img := range images {
					logrus.WithField("reponame", img.repoName).Infoln("Indexing image")
					tasks <- Parse(img.repoName, img.image)
				}
			}()
		}

		batch := idx.NewBatch()
		go func() {
			for task := range tasks {
				batch.Index(task.FullName, task)
			}
			if err := idx.Batch(batch); err != nil {
				logrus.WithError(err).Errorln("Failed to index initial repository state")
			}
			close(done)
		}()

		// When all images have been pushd to the channel, close it
		browseImgWg.Wait()
		close(images)

		// When all images have been parsed and submited to indexation batch channel, close it
		parseImgWg.Wait()
		close(tasks)

	}()
	return done
}

// GetImage returns the docker image ready to be indexed
func (idx *Index) GetImage(repository, tag string, dg digest.Digest) (*dim.IndexImage, error) {
	named, _ := reference.ParseNamed(repository)
	var repo dim.Repository
	var err error
	if repo, err = idx.RegClient.NewRepository(named); err != nil {
		logrus.WithError(err).WithField("Repository", repository).Errorln("Failed get repository info")
		return nil, err
	}

	var img *dim.RegistryImage
	if img, err = repo.ImageFromManifest(dg, tag); err != nil {
		logrus.WithError(err).Errorln("Failed to get image info from manifest")
		return nil, err
	}
	return Parse(repository, img), nil
}

// IndexImage adds a given image into the index
func (idx *Index) IndexImage(image *dim.IndexImage) {
	logrus.WithFields(logrus.Fields{"imageID": image.ID, "image.FullName": image.FullName}).Debugln("Indexing image")
	idx.Index.Index(image.FullName, image)
}

// DeleteImage removes an image from the index
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
	}

	for _, h := range sr.Hits {
		l.WithField("image.FullName", h.Fields["FullName"].(string)).Infoln("Removing image from index")
		idx.Index.Delete(h.Fields["FullName"].(string))
	}
}

// BuildQuery returns the query object corresponding to given parameters
func BuildQuery(nameTag, advanced string) bleve.Query {
	l := logrus.WithFields(logrus.Fields{"nameTag": nameTag, "advanced": advanced})
	l.Debugln("Building query clause")

	if nameTag == "*" || advanced == "*" {
		return bleve.NewMatchAllQuery()
	}

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

// SearchImages returns the images matching query.
// If fields is not empty, it fetches all given fields as well
func (idx *Index) SearchImages(q, a string, fields []string, offset, maxResults int) (*dim.IndexResults, error) {
	var err error
	var sr *bleve.SearchResult
	request := bleve.NewSearchRequestOptions(BuildQuery(q, a), maxResults, offset, false)
	request.Fields = []string{"Name", "Tag", "FullName", "Labels", "Envs"}
	l := logrus.WithField("request", request).WithField("query", request.Query)
	l.Debugln("Running search")
	if sr, err = idx.Search(request); err != nil {
		return nil, fmt.Errorf("Error occured when processing search : %v", err)
	}

	if fields != nil && len(fields) > 0 {
		detailFields := make([]string, len(fields))
		copy(detailFields, fields)
		for _, f := range []string{"Name", "Tag", "FullName"} {
			if !utils.ListContains(detailFields, f) {
				detailFields = append(detailFields, f)
			}
		}

		for i, h := range sr.Hits {
			if sr.Hits[i], err = idx.searchDetails(h, detailFields); err != nil {
				return nil, fmt.Errorf("Error occured while searching details of an image : %x", err)
			}
		}
	}

	results := &dim.IndexResults{Total: sr.Total}
	results.Images = buildResults(sr)

	return results, nil
}

func buildResults(sr *bleve.SearchResult) []*dim.IndexImage {
	images := make([]*dim.IndexImage, 0, sr.Total)
	for _, h := range sr.Hits {
		images = append(images, DocumentToImage(h))
	}
	return images
}

func (idx *Index) searchDetails(doc *search.DocumentMatch, fields []string) (*search.DocumentMatch, error) {
	logrus.WithField("doc", doc).WithField("fields", fields).Debugln("Entering searchDetails")
	request := bleve.NewSearchRequest(bleve.NewDocIDQuery([]string{doc.ID}))
	request.Fields = fields
	if doc.Fields["Labels"] != nil && utils.ListContains(fields, "Labels") {
		switch f := doc.Fields["Labels"].(type) {
		case string:
			request.Fields = append(request.Fields, fmt.Sprintf("Label.%s", f))
		case []interface{}:
			for _, f := range doc.Fields["Labels"].([]interface{}) {
				request.Fields = append(request.Fields, fmt.Sprintf("Label.%s", f))
			}
		}
	}
	if doc.Fields["Envs"] != nil && utils.ListContains(fields, "Envs") {
		switch f := doc.Fields["Envs"].(type) {
		case string:
			request.Fields = append(request.Fields, fmt.Sprintf("Env.%s", f))
		case []interface{}:
			for _, f := range doc.Fields["Envs"].([]interface{}) {
				request.Fields = append(request.Fields, fmt.Sprintf("Env.%s", f))
			}
		}
	}

	var sr *bleve.SearchResult
	var err error
	if sr, err = idx.Search(request); err != nil {
		return nil, fmt.Errorf("Failed to fetch all image info : %v", err)
	}

	return sr.Hits[0], err
}

// FindImage returns the image from the index with the given id
func (idx *Index) FindImage(id string) (*dim.IndexImage, error) {
	l := logrus.WithField("id", id)
	l.Debugln("Entering FindImage")
	q := bleve.NewTermQuery(id).SetField("ID")
	rq := bleve.NewSearchRequest(q)
	rq.Fields = []string{"ID", "Name", "FullName", "Tag", "Comment", "Created", "Author", "Label", "Labels", "Volumes", "ExposedPorts", "Env", "Envs", "Size"}

	var sr *bleve.SearchResult
	var err error
	if sr, err = idx.Search(rq); err != nil || sr.Total == 0 {
		return nil, fmt.Errorf("No image found for given id : %v", err)
	}
	if sr.Total > 1 {
		return nil, fmt.Errorf("Found many images for given id")
	}

	return DocumentToImage(sr.Hits[0]), nil
}

// DocumentToImage reads all fields of the given DocumentMatch and returns an image
func DocumentToImage(h *search.DocumentMatch) *dim.IndexImage {
	logrus.WithField("hit", h).Debugln("Entering documentToSearchResult")
	result := &dim.IndexImage{
		Name:     h.Fields["Name"].(string),
		Tag:      h.Fields["Tag"].(string),
		FullName: h.Fields["FullName"].(string),
	}

	if h.Fields["Created"] != nil {
		if t, err := time.Parse(time.RFC3339, h.Fields["Created"].(string)); err == nil {
			result.Created = t
		} else {
			logrus.WithError(err).WithField("time", h.Fields["Created"].(string)).Errorln("Failed to parse time")
		}
	}

	labels := make(map[string]string, 10)
	envs := make(map[string]string, 10)
	for k, v := range h.Fields {
		if strings.HasPrefix(k, "Label.") {
			labels[strings.TrimPrefix(k, "Label.")] = v.(string)
		} else if strings.HasPrefix(k, "Env.") {
			envs[strings.TrimPrefix(k, "Env.")] = v.(string)
		}
	}

	if len(labels) > 0 {
		result.Label = labels
	}
	if h.Fields["Volumes"] != nil {
		switch vol := h.Fields["Volumes"].(type) {
		case string:
			result.Volumes = []string{vol}
		case []interface{}:
			result.Volumes = make([]string, len(vol))
			for i, volume := range vol {
				result.Volumes[i] = volume.(string)
			}
		}
	}
	if h.Fields["ExposedPorts"] != nil {
		switch ports := h.Fields["ExposedPorts"].(type) {
		case float64:
			result.ExposedPorts = []int{int(ports)}
		case []interface{}:
			result.ExposedPorts = make([]int, len(ports))
			for i, port := range ports {
				result.ExposedPorts[i] = int(port.(float64))
			}
		}
	}
	if len(envs) > 0 {
		result.Env = envs
	}
	if h.Fields["Size"] != nil {
		result.Size = int64(h.Fields["Size"].(float64))
	}

	return result
}

//Submit pushes a NotificationJob that will be applied to the index
func (idx *Index) Submit(job *dim.NotificationJob) {
	idx.notifications <- job
}

func (idx *Index) loop(parallels int) {
	for i := 0; i < parallels; i++ {
		go idx.handleNotifications()
	}
}

func (idx *Index) handleNotifications() {
	for job := range idx.notifications {
		l := logrus.WithField("Event", job)

		hooks := idx.Config.GetHooks(job.Action)
		switch job.Action {
		case dim.DeleteAction:
			if len(hooks) > 0 {
				l.Debugln("Calling delete hooks")
				if img, err := idx.FindImage(job.Digest.String()); err == nil {
					triggerHooks(hooks, img)
				} else {
					l.WithError(err).Errorln("Failed to handle delete hook")
				}
			} else {
				l.Debugln("No delete hook found")
			}
			idx.DeleteImage(job.Digest.String())
		case dim.PushAction:
			if img, err := idx.GetImage(job.Repository, job.Tag, job.Digest); err == nil {
				if len(hooks) > 0 {
					l.Debugln("Calling delete hooks")
					triggerHooks(hooks, img)
				} else {
					l.Debugln("No push hook found")
				}
				idx.IndexImage(img)

			} else {
				logrus.WithField("Event", job).WithError(err).Errorln("Failed to handle push hook")
			}
		}
	}
}

func triggerHooks(hooks []*Hook, img *dim.IndexImage) {
	log := logrus.WithField("image", img)
	log.Debugln("Triggering hooks")
	for _, hook := range hooks {
		go func(h *Hook, i *dim.IndexImage) {
			if err := h.Eval(i); err != nil {
				log.WithError(err).Errorln("An error occured while processing hook")
			}
		}(hook, img)
	}
}
