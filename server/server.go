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

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"context"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/notifications"
	"github.com/mailgun/manners"
	"github.com/nhurel/dim/lib/environment"
	"github.com/nhurel/dim/lib/index"
	"github.com/nhurel/dim/types"
)

// Server type handle  indexation of a docker registry and serves the search endpoint
type Server struct {
	*manners.GracefulServer
	index *index.Index
}

// NewServer creates a new Server instance to listen on given port and use given index
func NewServer(port string, index *index.Index, ctx context.Context) *Server {
	c := environment.Set(ctx, environment.StartTimeKey, time.Now())
	http.HandleFunc("/v1/search", handler(index, Search))
	http.HandleFunc("/dim/notify", handler(index, NotifyImageChange))
	http.HandleFunc("/dim/version", buildVersionHandler(c))
	http.HandleFunc("/v2/_catalog", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{}")
	})
	return &Server{manners.NewWithServer(&http.Server{Addr: port, Handler: http.DefaultServeMux}), index}
}

// Run starts the server instance
func (s *Server) Run() error {
	return s.ListenAndServe()
}

// Handler injects an index into an HandlerFunc
func handler(i *index.Index, dhf DimHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dhf(i, w, r)
	}
}

func buildVersionHandler(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Version(ctx, w, r)
	}
}

// Version return server info including info and uptime
func Version(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	info := types.Info{}
	v := environment.Get(ctx, environment.VersionKey)
	if v != nil {
		info.Version = v.(string)
	}
	start := environment.Get(ctx, environment.StartTimeKey)
	if start != nil {
		info.Uptime = time.Now().Sub(start.(time.Time)).String()
	}
	if b, err := json.Marshal(info); err != nil {
		http.Error(w, "Failed to serialize the response", http.StatusInternalServerError)
		logrus.WithError(err).Errorln("Error occured while serializing server info")
	} else {
		fmt.Fprint(w, string(b))
	}
}

// NotifyImageChange handles docker registry events
func NotifyImageChange(i index.RegistryIndex, w http.ResponseWriter, r *http.Request) {

	logrus.Infoln("Receiving event from registry")
	defer r.Body.Close()

	enveloppe := &notifications.Envelope{}

	if err := json.NewDecoder(r.Body).Decode(enveloppe); err != nil {
		logrus.WithError(err).Errorln("Failed to parse event")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Notiifcation handled !")

	logrus.WithField("enveloppe", enveloppe).Debugln("Processing event")

	for _, event := range enveloppe.Events {
		switch event.Action {
		case notifications.EventActionDelete:
			i.DeleteImage(string(event.Target.Digest))
		case notifications.EventActionPush:
			if event.Target.MediaType == schema2.MediaTypeManifest {
				if err := i.GetImageAndIndex(event.Target.Repository, event.Target.Tag, event.Target.Digest); err != nil {
					logrus.WithField("EventTarget", event.Target).WithError(err).Errorln("Failed to reindex image")
				}
			} else {
				logrus.WithField("mediatype", event.Target.MediaType).WithField("Event", event).Debugln("Event safely ignored because mediatype is unknown")
			}
		default:
			logrus.WithField("Action", event.Action).WithField("Event", event).Debugln("Event safely ignored")
		}
	}
}

// Search handles docker search request
func Search(i index.RegistryIndex, w http.ResponseWriter, r *http.Request) {

	var err error
	var b []byte

	if err = r.ParseForm(); err != nil {
		logrus.WithError(err).Errorln("Failed to parse query")
		http.Error(w, "Failed to parse query", http.StatusBadRequest)
	}
	q, a, fields := r.Form.Get("q"), r.Form.Get("a"), r.Form["f"]

	// No error handling here. Using defaults if wrong params given
	offset, _ := strconv.Atoi(r.FormValue("offset"))
	maxResults, _ := strconv.Atoi(r.FormValue("maxResults"))
	if maxResults == 0 {
		maxResults = 10
	}

	if q == "" && a == "" {
		http.Error(w, "No search criteria provided", http.StatusBadRequest)
		return
	}

	var sr *bleve.SearchResult
	l := logrus.WithFields(logrus.Fields{"query": q, "advanced_query": a, "fields": fields})
	l.Debugln("Searching image")
	if sr, err = i.SearchImages(q, a, fields, offset, maxResults); err != nil {
		http.Error(w, "An error occured while procesing your request", http.StatusInternalServerError)
		l.WithError(err).Errorln("Error occured when processing search")
		return
	}

	results := types.SearchResults{NumResults: int(sr.Total), Query: q}
	l.WithField("#results", results.NumResults).Debugln("Found results")

	results.Results = buildResults(sr)

	if b, err = json.Marshal(results); err != nil {
		http.Error(w, "Failed to serialize the response", http.StatusInternalServerError)
		l.WithError(err).Errorln("Error occured while serializing search results")
	} else {
		fmt.Fprint(w, string(b))
	}
}

func buildResults(sr *bleve.SearchResult) []types.SearchResult {
	images := make([]types.SearchResult, 0, sr.Total)
	for _, h := range sr.Hits {
		images = append(images, documentToSearchResult(h))
	}
	return images
}

func documentToSearchResult(h *search.DocumentMatch) types.SearchResult {
	logrus.WithField("hit", h).Debugln("Entering documentToSearchResult")
	result := types.SearchResult{
		Name:        h.Fields["Name"].(string),
		Description: h.Fields["Tag"].(string),
		Tag:         h.Fields["Tag"].(string),
		FullName:    h.Fields["FullName"].(string),
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

// DimHandlerFunc injects index into a HandlerFunc function
type DimHandlerFunc func(i index.RegistryIndex, w http.ResponseWriter, r *http.Request)
