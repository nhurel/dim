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

package server_test

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"io/ioutil"

	"fmt"

	"bytes"

	"context"
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/notifications"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/environment"
	"github.com/nhurel/dim/lib/index"
	"github.com/nhurel/dim/lib/index/indextest"
	"github.com/nhurel/dim/lib/mock"
	"github.com/nhurel/dim/lib/utils"
	"github.com/nhurel/dim/server"
)

var (
	images = []dim.IndexImage{
		{
			ID:           "123456",
			Name:         "mongodb",
			Tag:          "latest",
			FullName:     "mongodb:latest",
			Created:      indextest.ParseTime("2016-07-24T09:05:06Z"),
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
			Created:  indextest.ParseTime("2016-06-23T09:05:06Z"),
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
			Created:      indextest.ParseTime("2016-06-30T09:05:06Z"),
			Env:          map[string]string{"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			Envs:         []string{"PATH"},
			ExposedPorts: []int{3306},
		},
		{
			ID:       "354678",
			Name:     "alpine",
			Tag:      "3.4",
			FullName: "alpine:3.4",
			Created:  indextest.ParseTime("2016-06-30T09:05:06Z"),
		},
	}
)

func TestSearch(t *testing.T) {

	var i bleve.Index
	var err error
	if i, err = indextest.MockIndex(index.ImageMapping); err != nil {
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
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/search?a=Name:%s&f=Volumes&f=Env&f=Label&f=ExposedPorts&f=Created", image.Name), nil)

		server.Search(ind, response, request)

		if response.Result().StatusCode != http.StatusOK {
			t.Fatalf("Expected status code 200 : %s", response.Result().Status)
		}

		var body []byte
		body, err = ioutil.ReadAll(response.Body)
		sr := &dim.SearchResults{}
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

		if image.Created != sr.Results[0].Created {
			t.Errorf("Expected created to be %v but got %v", image.Created, sr.Results[0].Created)
		}

	}

	request := httptest.NewRequest(http.MethodGet, "/v1/search?f=Name", nil)
	response := httptest.NewRecorder()
	server.Search(ind, response, request)
	if response.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("Search returned status %s instead of %d when called with no search param", response.Result().Status, http.StatusBadRequest)
	}
}

func TestNotifyImageChange(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	ind := &mock.NoOpRegistryIndex{Calls: make(map[string][]interface{})}

	deleteEventDigest := digest.NewDigestFromBytes(digest.Canonical, []byte("delete"))
	deleteEvent := notifications.Event{
		Action: notifications.EventActionDelete,
	}
	deleteEvent.Target.Descriptor = distribution.Descriptor{Digest: deleteEventDigest}

	deleteEventMessage, err := json.Marshal(&notifications.Envelope{Events: []notifications.Event{deleteEvent}})

	if err != nil {
		t.Fatalf("Failed to create tests : %v", err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/dim/notify", bytes.NewBuffer(deleteEventMessage))

	server.NotifyImageChange(ind, response, request)

	if ind.Calls["Submit"] == nil || ind.Calls["Submit"][0].(*dim.NotificationJob).Action != dim.DeleteAction {
		t.Errorf("NotifyImageChange(deleteEvent) did not submit a job with action %s. Called with %v", dim.DeleteAction, ind.Calls["Submit"][0])
	}

	pushEventDigest := digest.NewDigestFromBytes(digest.Canonical, []byte("push"))
	pushEvent := notifications.Event{
		Action: notifications.EventActionPush,
	}

	pushEvent.Target.Repository = "pushRepository"
	pushEvent.Target.Tag = "latest"
	pushEvent.Target.Descriptor = distribution.Descriptor{
		Digest:    pushEventDigest,
		MediaType: schema2.MediaTypeManifest,
	}

	pushEventMessage, err := json.Marshal(&notifications.Envelope{Events: []notifications.Event{pushEvent}})

	if err != nil {
		t.Fatalf("Failed to create tests : %v", err)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/dim/notify", bytes.NewBuffer(pushEventMessage))

	server.NotifyImageChange(ind, response, request)

	if ind.Calls["Submit"] == nil {
		t.Errorf("NotifyImageChange(pushEvent) did not call index.GetImageAndIndex %v", ind.Calls)
	} else {
		j := ind.Calls["Submit"][0].(*dim.NotificationJob)
		if j.Action != dim.PushAction ||
			j.Repository != pushEvent.Target.Repository || // checking repository param
			j.Tag != pushEvent.Target.Tag || // checking tag param
			j.Digest != pushEventDigest { // checking digest param
			t.Errorf("NotifyImageChange(pushEvent) called index.Submit with %v instead of  %s, %s, %s, %v", ind.Calls["GetImageAndIndex"], dim.PushAction, pushEvent.Target.Repository, pushEvent.Target.Tag, pushEventDigest)
		}
	}

	badMessage := ""
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/dim/notify", bytes.NewBufferString(badMessage))

	server.NotifyImageChange(ind, response, request)
	if response.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("NotifyImaeChange(badRequest) returned status %s instead of %d", response.Result().Status, http.StatusBadRequest)
	}

}

func TestVersion(t *testing.T) {

	scenarii := []struct {
		given    context.Context
		expected dim.Info
	}{
		{
			given:    context.Background(),
			expected: dim.Info{Version: ""},
		},
		{
			given:    environment.Set(context.Background(), environment.VersionKey, "1.0.0"),
			expected: dim.Info{Version: "1.0.0"},
		},
		{
			given:    environment.Set(environment.Set(context.Background(), environment.VersionKey, "1.0.0"), environment.StartTimeKey, time.Now()),
			expected: dim.Info{Version: "1.0.0", Uptime: "100ms"},
		},
	}
	for _, scenario := range scenarii {
		w := httptest.NewRecorder()
		server.Version(scenario.given, w, nil)

		got := &dim.Info{}
		b, err := ioutil.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("Failed to parse response : %v", err)
		}
		if err := json.Unmarshal(b, got); err != nil {
			t.Fatalf("Failed to parse response : %v", err)
		}
		if got.Version != scenario.expected.Version {
			t.Errorf("/dim/version returned %s instead of %s", got.Version, scenario.expected.Version)
		}
		if scenario.expected.Uptime == "" && got.Uptime != "" {
			t.Errorf("/dim/version return an unexpected uptime. Got %s - Expected %s", got.Uptime, scenario.expected.Uptime)
		}
		if scenario.expected.Uptime != "" {
			if _, err := time.ParseDuration(got.Uptime); err != nil {
				t.Errorf("/dim/version returned  a wrong uptime %s : %v", got.Uptime, err)
			}

		}
	}

}
