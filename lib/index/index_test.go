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

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/index/indextest"
	. "gopkg.in/check.v1"
)

type TestSuite struct {
	index *Index
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&TestSuite{})

var (
	images = []dim.IndexImage{
		{
			ID:       "123456",
			Name:     "centos",
			Tag:      "centos6",
			FullName: "centos:centos6",
			Created:  indextest.ParseTime("2016-07-24T09:05:06Z"),
			Label: map[string]string{
				"type":   "base",
				"family": "rhel",
			},
			Labels: []string{
				"type",
				"family",
			},
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
			ID:       "354678",
			Name:     "mysql",
			Tag:      "5.7",
			FullName: "mysql:5.7",
			Created:  indextest.ParseTime("2016-06-30T09:05:06Z"),
			Label: map[string]string{
				"type":      "sql",
				"family":    "debian",
				"framework": "mysql",
			},
			Labels: []string{
				"type",
				"family",
				"framework",
			},
			Volumes: []string{"/var/lib/mysql"},
			Env: map[string]string{
				"PATH":          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"MYSQL_MAJOR":   "5.7",
				"MYSQL_VERSION": "5.7.9-1debian8",
			},
			Envs: []string{
				"PATH",
				"MYSQL_MAJOR",
				"MYSQL_VERSION",
			},
			ExposedPorts: []int{3306},
		},
	}
)

func (s *TestSuite) SetUpSuite(c *C) {
	logrus.SetLevel(logrus.InfoLevel)
	var i bleve.Index
	var err error
	if i, err = indextest.MockIndex(ImageMapping); err != nil {
		logrus.WithError(err).Errorln("Failed to create index")
		return
	}

	s.index = &Index{Index: i, RegClient: nil, Config: &Config{}}
}

func (s *TestSuite) SetUpTest(c *C) {
	for _, image := range images {
		if err := s.index.Index.Index(image.FullName, image); err != nil {
			logrus.WithError(err).Errorln("Failed to index image")
		}
	}
}

func (s *TestSuite) TearDownSuite(c *C) {
	s.index.Index.Close()
}

func (s *TestSuite) TestNameTagSearch(c *C) {
	var tests = []struct {
		query       string
		resultNames []string
	}{
		{"mysql", []string{"mysql"}},
		{"ysql", []string{"mysql"}},
		{"sql", []string{"mysql"}},
		{"5.7", []string{"mysql"}},
		{"5.7.9", []string{}},
		{"1debian8", []string{}},
		{"debian", []string{}},
		{"base", []string{}},
		{"apache2", []string{}},
		{"mysql:5.7", []string{"mysql"}},
		{"http*:2.4", []string{"httpd"}},
		{"*", []string{"centos", "httpd", "mysql"}},
	}

	for _, t := range tests {
		c.Logf("Test with query %s", t)
		request := bleve.NewSearchRequest(BuildQuery(t.query, ""))
		request.Fields = []string{"Name", "Tag"}
		results, err := s.index.Search(request)
		c.Assert(err, IsNil)
		c.Log(results)
		c.Assert(results.Total, Equals, uint64(len(t.resultNames)))
		c.Assert(results.Hits, HasLen, len(t.resultNames))
		for i, r := range t.resultNames {
			c.Assert(results.Hits[i].Fields["Name"], Equals, r)
		}
	}
}

func (s *TestSuite) TestAdvancedSearch(c *C) {
	var tests = []struct {
		query       string
		resultNames []string
	}{
		{"Name:mysql", []string{"mysql"}},
		{"Tag:5.7", []string{"mysql"}},
		{"Env.MYSQL_VERSION:5.7.9-1debian8", []string{"mysql"}},
		{"Env.MYSQL_VERSION:5.7.9", []string{"mysql"}},
		{"Label.family:debian", []string{"httpd", "mysql"}},
		{"Label.type:base", []string{"centos"}},
		{"Labels:type", []string{"centos", "httpd", "mysql"}},
		{"Labels:/frame.*/", []string{"httpd", "mysql"}},
		{"Labels:frame*", []string{"httpd", "mysql"}},
		{"Env.HTTPD_VERSION:2*", []string{"httpd"}},
		{"Env.HTTPD_VERSION:2.*", []string{"httpd"}},
		{"Env.HTTPD_VERSION:/*/", []string{"httpd"}},
		{"Envs:HTTPD_PREFIX", []string{"httpd"}},
		{"Envs:HTTP*", []string{"httpd"}},
		{"apache", []string{"httpd"}},
		{"*", []string{"centos", "httpd", "mysql"}},
		{"+Created:>\"2016-07-01T00:00:00Z\"", []string{"centos"}},
		{"+Created:<\"2016-06-24T00:00:00Z\"", []string{"httpd"}},
	}

	for _, t := range tests {
		c.Logf("Test with query %s", t)
		request := bleve.NewSearchRequest(BuildQuery("", t.query))
		request.Fields = []string{"Name", "Tag"}
		results, err := s.index.Search(request)
		c.Assert(err, IsNil)
		c.Log(results)
		c.Assert(results.Total, Equals, uint64(len(t.resultNames)))
		c.Assert(results.Hits, HasLen, len(t.resultNames))
		for i, r := range t.resultNames {
			c.Assert(results.Hits[i].Fields["Name"], Equals, r)
		}
	}
}

func (s *TestSuite) TestSearchResults(c *C) {
	request := bleve.NewSearchRequest(BuildQuery("mysql", ""))
	request.Fields = []string{"Name", "Tag", "ExposedPorts", "Volumes", "Labels", "Envs"}
	results, err := s.index.Search(request)
	c.Assert(err, IsNil)
	c.Log(results)
	c.Assert(results.Total, Equals, uint64(1))
	c.Assert(results.Hits, HasLen, 1)
	c.Assert(results.Hits[0].Fields["ExposedPorts"], Equals, float64(3306))
	c.Assert(results.Hits[0].Fields["Tag"], Equals, "5.7")
	c.Assert(results.Hits[0].Fields["Volumes"], DeepEquals, "/var/lib/mysql")
	c.Assert(results.Hits[0].Fields["Labels"], DeepEquals, []interface{}{"type", "family", "framework"})
	c.Assert(results.Hits[0].Fields["Envs"], DeepEquals, []interface{}{"PATH", "MYSQL_MAJOR", "MYSQL_VERSION"})

}

func (s *TestSuite) TestDeleteImage(c *C) {
	for _, image := range images {
		request := bleve.NewSearchRequest(BuildQuery("", fmt.Sprintf("+Name:%s +Tag:%s", image.Name, image.Tag)))
		request.Fields = []string{"Name", "Tag"}
		results, err := s.index.Search(request)
		c.Assert(err, IsNil)
		c.Assert(results.Hits, HasLen, 1)
		s.index.DeleteImage(image.ID)
		results, err = s.index.Search(request)
		c.Assert(err, IsNil)
		c.Assert(results.Hits, HasLen, 0)
	}
}
