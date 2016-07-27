package index

import (
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	. "gopkg.in/check.v1"
	"path"
	"testing"
	"time"
)

type TestSuite struct {
	index *Index
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&TestSuite{})

var (
	images = []Image{
		Image{
			ID:      "123456",
			Name:    "centos",
			Tag:     "centos6",
			Created: parseTime("2016-07-24T09:05:06"),
			Labels: map[string]interface{}{
				"type":   "base",
				"family": "rhel",
			},
		},
		Image{
			ID:      "234567",
			Name:    "httpd",
			Tag:     "2.4",
			Created: parseTime("2016-06-23T09:05:06"),
			Labels: map[string]interface{}{
				"type":      "web",
				"family":    "debian",
				"framework": "apache-httpd",
			},
			Volumes: []string{"/var/www/html"},
			Env: map[string]string{
				"PATH=/usr/local/sbin:/usr/local/bin": "/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/apache2/bin",
				"HTTPD_PREFIX":                        "/usr/local/apache2",
				"HTTPD_VERSION":                       "2.4.18",
				"HTTPD_BZ2_URL":                       "https://www.apache.org/dist/httpd/httpd-2.4.18.tar.bz2",
			},
			ExposedPorts: []int{80, 443},
		},
		Image{
			ID:      "354678",
			Name:    "mysql",
			Tag:     "5.7",
			Created: parseTime("2016-06-30T09:05:06"),
			Labels: map[string]interface{}{
				"type":      "sql",
				"family":    "debian",
				"framework": "mysql",
			},
			Volumes: []string{"/var/lib/mysql"},
			Env: map[string]string{
				"PATH":          "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				"MYSQL_MAJOR":   "5.7",
				"MYSQL_VERSION": "5.7.9-1debian8",
			},
			ExposedPorts: []int{3306},
		},
	}
)

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

func (s *TestSuite) SetUpSuite(c *C) {
	logrus.SetLevel(logrus.InfoLevel)
	dir := path.Join("test.index", time.Now().Format("20060102150405.000"))
	var i bleve.Index
	var err error

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", imageMapping)
	mapping.DefaultField = "_all"
	if i, err = bleve.New(dir, mapping); err != nil {
		logrus.WithError(err).Errorln("Failed to create index")
		return
	}
	s.index = &Index{i, "", nil, nil}

	for _, image := range images {
		if err := s.index.Index.Index(image.ID, image); err != nil {
			logrus.WithError(err).Errorln("Failed to index image")
		}
	}

}

func (s *TestSuite) SetUpTest(c *C) {
	// Nothing
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
	}

	for _, t := range tests {
		c.Logf("Test with query %s", t)
		request := bleve.NewSearchRequest(s.index.BuildQuery(t.query, ""))
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
		{"Labels.family:debian", []string{"httpd", "mysql"}},
		{"Labels.type:base", []string{"centos"}},
		{"Env.HTTPD_VERSION:2*", []string{"httpd"}},
		{"Env.HTTPD_VERSION:2.*", []string{"httpd"}},
		{"Env.HTTPD_VERSION:/*/", []string{"httpd"}},
		{"apache", []string{"httpd"}},
	}

	for _, t := range tests {
		c.Logf("Test with query %s", t)
		request := bleve.NewSearchRequest(s.index.BuildQuery("", t.query))
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
