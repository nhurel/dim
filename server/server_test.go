package server_test

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"io/ioutil"

	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/nhurel/dim/lib/index"
	"github.com/nhurel/dim/lib/index/indextest"
	"github.com/nhurel/dim/lib/utils"
	"github.com/nhurel/dim/server"
	"github.com/nhurel/dim/types"
	"gopkg.in/square/go-jose.v1/json"
)

var (
	images = []index.Image{
		{
			ID:           "123456",
			Name:         "mongodb",
			Tag:          "latest",
			FullName:     "mongodb:latest",
			Created:      indextest.ParseTime("2016-07-24T09:05:06"),
			Volumes:      []string{"/data/configdb", "/data/db"},
			ExposedPorts: []int{27017},
			Env: map[string]string{
				"PATH":          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"GOSU_VERSION":  "1.7",
				"GPG_KEYS":      "DFFA3DCF326E302C4787673A01C4E7FAAAB2461C      42F3E95A2C4F08279C4960ADD68FA50FEA312927",
				"MONGO_MAJOR":   "3.2",
				"MONGO_VERSION": "3.2.10",
				"MONGO_PACKAGE": "mongodb-org",
			},
			Envs: []string{"PATH", "GOSU_VERSION", "GPG_KEYS", "MONGO_MAJOR", "MONGO_VERSION", "MONGO_PACKAGE"},
		},
		{
			ID:       "234567",
			Name:     "httpd",
			Tag:      "2.4",
			FullName: "httpd:2.4",
			Created:  indextest.ParseTime("2016-06-23T09:05:06"),
			Label: map[string]string{
				"type":      "web",
				"family":    "debian",
				"framework": "apache-httpd",
			},
			Labels: []string{
				"type",
				"family",
				"framework",
			},
			Volumes: []string{"/var/www/html"},
			Env: map[string]string{
				"PATH":          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/apache2/bin",
				"HTTPD_PREFIX":  "/usr/local/apache2",
				"HTTPD_VERSION": "2.4.18",
				"HTTPD_BZ2_URL": "https://www.apache.org/dist/httpd/httpd-2.4.18.tar.bz2",
			},
			Envs: []string{
				"PATH",
				"HTTPD_PREFIX",
				"HTTPD_VERSION",
				"HTTPD_BZ2_URL",
			},
			ExposedPorts: []int{80, 443},
		},
		{
			ID:           "354678",
			Name:         "debian",
			Tag:          "wheezy",
			FullName:     "debian:wheezy",
			Created:      indextest.ParseTime("2016-06-30T09:05:06"),
			Env:          map[string]string{"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			Envs:         []string{"PATH"},
			ExposedPorts: []int{3306},
		},
		{
			ID:       "354678",
			Name:     "alpine",
			Tag:      "3.4",
			FullName: "alpine:3.4",
			Created:  indextest.ParseTime("2016-06-30T09:05:06"),
		},
	}
)

func TestSearch(t *testing.T) {

	var i bleve.Index
	var err error
	if i, err = indextest.MockIndex(); err != nil {
		t.Fatalf("Failed to create index : %v", err)
	}

	ind := &index.Index{Index: i}

	for _, image := range images {
		if err := ind.Index.Index(image.FullName, image); err != nil {
			logrus.WithError(err).Errorln("Failed to index image")
		}
	}

	for _, image := range images {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/search?a=Name:%s&f=Volumes&f=Env&f=Label&f=ExposedPorts", image.Name), nil)

		server.Search(ind, response, request)

		if response.Result().StatusCode != http.StatusOK {
			t.Fatalf("Expected status code 200 : %s", response.Result().Status)
		}

		var body []byte
		body, err = ioutil.ReadAll(response.Body)
		sr := &types.SearchResults{}
		json.Unmarshal(body, sr)
		if sr.NumResults != 1 {
			t.Fatalf("Image %s not found : %s", image.FullName, string(body))
		}
		logrus.Errorln(string(body))
		if utils.MapMatchesAll(sr.Results[0].Env, image.Env) {
			t.Errorf("Env found %v is different from environment expected %v", sr.Results[0].Env, image.Env)
		}
		if utils.MapMatchesAll(sr.Results[0].Label, image.Label) {
			t.Errorf("Labels found %v is different from labels expected %v", sr.Results[0].Label, image.Label)
		}

		for _, ev := range image.Volumes {
			if !utils.ListContains(sr.Results[0].Volumes, ev) {
				t.Errorf("Expected volume %s not found in volumes returned %v", ev, sr.Results[0].Volumes)
			}
		}

		for _, ep := range image.ExposedPorts {
			var ok = false
			for _, gp := range sr.Results[0].ExposedPorts {
				if ep == gp {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("Expected port %d not found in %v", ep, sr.Results[0].ExposedPorts)
			}
		}

	}

}
