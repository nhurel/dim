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
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/image"
	"github.com/docker/go-connections/nat"
	"github.com/nhurel/dim/lib"
	. "gopkg.in/check.v1"
)

type ImageTestSuite struct {
}

// Hook up gocheck into the "go test" runner.
func TestImage(t *testing.T) { TestingT(t) }

var _ = Suite(&ImageTestSuite{})

var img = &dim.RegistryImage{

	Image: &image.Image{
		V1Image: image.V1Image{
			ID:      "imageID",
			Parent:  "alpine:latest",
			Comment: "comment",
			Created: time.Now(),
			Author:  "authorName",
			Config: &container.Config{
				Env: []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/apache2/bin",
					"HTTPD_PREFIX=usr/local/apache2",
					"HTTPD_VERSION=2.4.18",
					"HTTPD_BZ2_URL=https://www.apache.org/dist/httpd/httpd-2.4.18.tar.bz2"},
				Volumes: map[string]struct{}{
					"/var/www/html": {},
				},
				Labels: map[string]string{
					"label1":              "value1",
					"label2.three.levels": "value2",
					"label3_2levels":      "value3",
				},
				ExposedPorts: map[nat.Port]struct{}{
					nat.Port("80"):  {},
					nat.Port("443"): {},
				},
			},
			Size: int64(2048),
		},
	},
	Tag:    "latest",
	Digest: "imageDigest",
}

func (s *ImageTestSuite) TestParse(c *C) {
	parsed := Parse("httpd", img)
	c.Assert(parsed.ExposedPorts, HasLen, 2)
	c.Assert(parsed.FullName, Equals, "httpd:latest")
	c.Assert(SliceContains(parsed.ExposedPorts, 80), Equals, true)
	c.Assert(SliceContains(parsed.ExposedPorts, 443), Equals, true)
	c.Assert(parsed.Author, Equals, img.Author)
	c.Assert(parsed.Comment, Equals, img.Comment)
	c.Assert(parsed.ID, Equals, img.Digest)
	c.Assert(parsed.Envs, HasLen, 4)
	c.Assert(parsed.Env, HasLen, 4)
	c.Assert(parsed.Env["PATH"], Equals, "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/apache2/bin")
	c.Assert(parsed.Env["HTTPD_PREFIX"], Equals, "usr/local/apache2")
	c.Assert(parsed.Env["HTTPD_VERSION"], Equals, "2.4.18")
	c.Assert(parsed.Env["HTTPD_BZ2_URL"], Equals, "https://www.apache.org/dist/httpd/httpd-2.4.18.tar.bz2")
	c.Assert(parsed.Volumes, HasLen, 1)
	c.Assert(parsed.Volumes[0], Equals, "/var/www/html")
	c.Assert(parsed.Labels, HasLen, 3)
	c.Assert(parsed.Label, HasLen, 3)
	c.Assert(parsed.Label["label1"], Equals, "value1")
	c.Assert(parsed.Label["label2.three.levels"], Equals, "value2")
	c.Assert(parsed.Label["label3_2levels"], Equals, "value3")
	c.Assert(parsed.Size, Equals, img.Size)
}

func SliceContains(s []int, c int) bool {
	for _, e := range s {
		if e == c {
			return true
		}
	}
	return false
}
