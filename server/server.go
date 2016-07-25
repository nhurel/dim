package server

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/docker/engine-api/types/registry"
	"github.com/mailgun/manners"
	"github.com/nhurel/dim/lib/index"
	"net/http"
)

type Server struct {
	*manners.GracefulServer
	index *index.Index
}

func NewServer(port string, index *index.Index) *Server {
	http.HandleFunc("/v1/search", handler(index, Search))
	return &Server{manners.NewWithServer(&http.Server{Addr: port, Handler: http.DefaultServeMux}), index}
}

func (s *Server) Run() error {
	return s.ListenAndServe()
}

// Handler injects an index into an HandlerFunc
func handler(i *index.Index, dhf DimHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dhf(i, w, r)
	}
}

// Search handles docker search request
func Search(i *index.Index, w http.ResponseWriter, r *http.Request) {

	var err error
	var b []byte

	q := r.FormValue("q")

	a := r.FormValue("a")

	if q == "" && a == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	var sr *bleve.SearchResult

	request := bleve.NewSearchRequest(i.BuildQuery(q, a))
	request.Fields = []string{"Name", "Tag"}
	logrus.WithField("request", request).WithField("query", request.Query).Debugln("Running search")
	if sr, err = i.Search(request); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}

	results := registry.SearchResults{NumResults: int(sr.Total), Query: q}
	images := make([]registry.SearchResult, 0, sr.Total)
	for _, h := range sr.Hits {
		images = append(images, documentToSearchResult(h))
	}
	results.Results = images

	if b, err = json.Marshal(results); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
	} else {
		fmt.Fprint(w, string(b))
	}
}

func documentToSearchResult(h *search.DocumentMatch) registry.SearchResult {
	logrus.WithField("hit", h).Debugln("Entering documentToSearchResult")
	result := registry.SearchResult{Name: h.Fields["Name"].(string), Description: h.Fields["Tag"].(string)}
	return result
}

// Use to inject index into a HandlerFunc function
type DimHandlerFunc func(i *index.Index, w http.ResponseWriter, r *http.Request)
