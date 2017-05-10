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
	"time"

	"context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/notifications"
	"github.com/mailgun/manners"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/environment"
)

// Server type handle  indexation of a docker registry and serves the search endpoint
type Server struct {
	*manners.GracefulServer
	index dim.RegistryIndex
}

// NewServer creates a new Server instance to listen on given port and use given index
func NewServer(cfg *Config, index dim.RegistryIndex, ctx context.Context, proxy dim.RegistryProxy) *Server {
	c := environment.Set(ctx, environment.StartTimeKey, time.Now())

	http.HandleFunc("/v1/search", securityFilter(cfg, handler(index, Search)))
	http.HandleFunc("/dim/notify", securityFilter(cfg, handler(index, NotifyImageChange)))
	http.HandleFunc("/dim/version", securityFilter(cfg, buildVersionHandler(c)))
	http.HandleFunc("/", securityFilter(cfg, proxy.Forwards))
	return &Server{manners.NewWithServer(&http.Server{Addr: cfg.Port, Handler: http.DefaultServeMux}), index}
}

// Run starts the server instance
func (s *Server) Run() error {
	return s.ListenAndServe()
}

// RunSecure starts the server instance in HTTPS
func (s *Server) RunSecure(certFile, keyFIile string) error {
	return s.ListenAndServeTLS(certFile, keyFIile)
}

// Handler injects an index into an HandlerFunc
func handler(i dim.RegistryIndex, dhf DimHandlerFunc) http.HandlerFunc {
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
	info := dim.Info{}
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
		w.Write(b)
	}
}

// NotifyImageChange handles docker registry events
func NotifyImageChange(i dim.RegistryIndex, w http.ResponseWriter, r *http.Request) {

	logrus.Debugln("Receiving event from registry")
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
			logrus.WithField("enveloppe", enveloppe).Infoln("Processing delete event")
			i.Submit(&dim.NotificationJob{Action: dim.DeleteAction, Digest: event.Target.Digest})
		case notifications.EventActionPush:
			if event.Target.MediaType == schema2.MediaTypeManifest {
				logrus.WithField("enveloppe", enveloppe).Infoln("Processing push event")
				i.Submit(&dim.NotificationJob{Action: dim.PushAction, Repository: event.Target.Repository, Tag: event.Target.Tag, Digest: event.Target.Digest})
			} else {
				logrus.WithField("mediatype", event.Target.MediaType).WithField("Event", event).Debugln("Event safely ignored because mediatype is unknown")
			}
		default:
			logrus.WithField("Action", event.Action).WithField("Event", event).Debugln("Event safely ignored")
		}
	}
}

// Search handles docker search request
func Search(i dim.RegistryIndex, w http.ResponseWriter, r *http.Request) {

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

	var sr *dim.IndexResults
	l := logrus.WithFields(logrus.Fields{"query": q, "advanced_query": a, "fields": fields})
	l.Debugln("Searching image")
	if sr, err = i.SearchImages(q, a, fields, offset, maxResults); err != nil {
		http.Error(w, "An error occured while procesing your request", http.StatusInternalServerError)
		l.WithError(err).Errorln("Error occured when processing search")
		return
	}

	results := dim.SearchResults{NumResults: int(sr.Total), Query: q}
	l.WithField("#results", results.NumResults).Debugln("Found results")

	results.Results = buildResults(sr)

	if b, err = json.Marshal(results); err != nil {
		http.Error(w, "Failed to serialize the response", http.StatusInternalServerError)
		l.WithError(err).Errorln("Error occured while serializing search results")
	} else {
		fmt.Fprint(w, string(b))
	}
}

func buildResults(sr *dim.IndexResults) []dim.SearchResult {
	images := make([]dim.SearchResult, 0, sr.Total)
	for _, i := range sr.Images {
		images = append(images, imageToSearchResult(i))
	}
	return images
}

func imageToSearchResult(i *dim.IndexImage) dim.SearchResult {
	logrus.WithField("image", i).Debugln("Entering imageToSearchResult")
	result := dim.SearchResult{
		Name:         i.Name,
		Description:  i.Tag,
		Tag:          i.Tag,
		FullName:     i.FullName,
		Created:      i.Created,
		Label:        i.Label,
		Volumes:      i.Volumes,
		ExposedPorts: i.ExposedPorts,
		Env:          i.Env,
		Size:         i.Size,
	}

	return result
}

// DimHandlerFunc injects index into a HandlerFunc function
type DimHandlerFunc func(i dim.RegistryIndex, w http.ResponseWriter, r *http.Request)
