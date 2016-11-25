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

package integration

import (
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/wrapper/dockerClient"
	. "gopkg.in/check.v1"
)

type IntegrationTestSuite struct {
	Dim *dim.Dim
}

// Hook up gocheck into the "go test" runner.
func TestIntegration(t *testing.T) { TestingT(t) }

var _ = Suite(&IntegrationTestSuite{})

type testImages struct {
	tag        string
	labels     []string
	labelsName []string
	volumes    []string
}

var integrationLabel = testImages{
	tag: "localhost/httpd:first",
	labels: []string{
		"type=web",
		"first=true",
		"framework=apache",
	},
	labelsName: []string{
		"type",
		"first",
		"framework",
	},
}

var dimExec = "../dim"

func runCommand(command string, args ...interface{}) (string, error) {
	params := make([]string, 0, len(args))

	for _, a := range args {
		switch a := a.(type) {
		case []string:
			params = append(params, a...)
		case string:
			params = append(params, a)
		}
	}

	o, err := exec.Command(command, params...).CombinedOutput()
	return string(o), err

}

func (s *IntegrationTestSuite) SetUpSuite(c *C) {
	s.Dim = &dim.Dim{Docker: &dockerClient.DockerClient{Insecure: true}}
	if err := s.Dim.Pull("httpd:2.4-alpine"); err != nil {
		c.Error(err)
	}
}

func (s *IntegrationTestSuite) TestLabelAndSearch(c *C) {
	if o, err := runCommand(dimExec, "label", "httpd:2.4-alpine", "-t", integrationLabel.tag, "-p", "-k", "-r", integrationLabel.labels); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	queries := []string{"Label.type:web", "Label.first:true"}

	for _, q := range queries {
		c.Logf("Checking query %s returns httpd:first\n", q)
		result, err := runCommand(dimExec, "search", "--registry-url=localhost", "-k", "-a", q, "-l debug")
		c.Log(result)
		if err != nil {
			c.Log(string(err.(*exec.ExitError).Stderr))
		}
		c.Assert(err, IsNil)
		re := regexp.MustCompile("httpd\\s*first.*first=true, framework=apache, type=web\\s*")
		c.Assert(re.MatchString(string(result)), Equals, true)
	}

}

func (s *IntegrationTestSuite) TestUnlabelAndSearch(c *C) {
	if o, err := runCommand(dimExec, "label", "httpd:2.4-alpine", "-t", integrationLabel.tag, "-p", integrationLabel.labels); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	if o, err := runCommand(dimExec, "label", "-d", "-o", "-r", integrationLabel.tag, "-p", "-k", integrationLabel.labelsName); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	queries := []string{"Label.type:web", "Label.first:true", "Labels.type:/*/"}
	for _, q := range queries {
		c.Logf("Checking query %s returns 0 result\n", q)
		result, err := runCommand(dimExec, "search", "--registry-url=localhost", "-k", "-a", q)
		c.Log(result)
		if err != nil {
			c.Error(string(err.(*exec.ExitError).Stderr))
		}
		c.Assert(err, IsNil)
		re := regexp.MustCompile("No result found")
		//c.Assert(string(result), Matches, "\n.*0 Results found.*")
		c.Assert(re.MatchString(string(result)), Equals, true)
	}
}

func (s *IntegrationTestSuite) TestDeleteAndSearch(c *C) {
	if o, err := runCommand(dimExec, "label", "-p", "httpd:2.4-alpine", "-t", integrationLabel.tag, "-r", "-k", integrationLabel.labels); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	if o, err := runCommand(dimExec, "delete", "--registry-url=localhost", "-k", "-r", integrationLabel.tag); err != nil {
		c.Log(o)
		c.Log(string(err.(*exec.ExitError).Stderr))
		c.Error("Error when deleting image")
	}
	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	result, err := runCommand(dimExec, "search", "--registry-url=localhost", "-k", "httpd")
	c.Log(result)
	if err != nil {
		c.Log(string(err.(*exec.ExitError).Stderr))
	}
	c.Assert(err, IsNil)
	re := regexp.MustCompile("No result found")
	//c.Assert(string(result), Matches, "\n.*0 Results found.*")
	c.Assert(re.MatchString(string(result)), Equals, true)

}

func (s *IntegrationTestSuite) TestVolumeOutput(c *C) {
	if o, err := runCommand(dimExec, "label", "-p", "redis:3.2.1-alpine", "-t", "localhost/redis:3.2.1-alpine", "-r", "-k", "type=database"); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	result, err := runCommand(dimExec, "search", "--registry-url=localhost", "-k", "redis")
	c.Log(result)
	if err != nil {
		c.Log(string(err.(*exec.ExitError).Stderr))
	}
	c.Assert(err, IsNil)
	re := regexp.MustCompile("redis\\s*3.2.1-alpine.*[/data]")
	c.Assert(re.MatchString(string(result)), Equals, true)
}

func (s *IntegrationTestSuite) TestShowCommand(c *C) {
	if o, err := runCommand(dimExec, "label", "-p", "redis:3.2.1-alpine", "-t", "localhost/redis:3.2.1-alpine", "-r", "-k", "type=database"); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	result, err := runCommand(dimExec, "show", "localhost/redis:3.2.1-alpine", "-k")

	c.Assert(err, IsNil)
	c.Assert(result, Equals, `Name :  localhost/redis:3.2.1-alpine
Id :  sha256:d65f1dcf63b7475dd45368a0bbabbd67be61598a02a37815b6e9fcfcfbf67d14
Labels:
type = database

Tags:
localhost/redis:3.2.1-alpine

Ports :
6379/tcp = {}

Volumes:
/data = {}

Env :
 PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
 REDIS_VERSION=3.2.1
 REDIS_DOWNLOAD_URL=http://download.redis.io/releases/redis-3.2.1.tar.gz
 REDIS_DOWNLOAD_SHA1=26c0fc282369121b4e278523fce122910b65fbbf

Entrypoint : [docker-entrypoint.sh]
Command : [redis-server]
`)

	_, err = runCommand(dimExec, "show", "localhost/redis:3.2.1-alpine", "-k", "-o", "show_test.out")
	c.Assert(err, IsNil)

	f, err := os.Open("show_test.out")
	c.Assert(err, IsNil)
	defer os.Remove(f.Name())
	fc, err := ioutil.ReadAll(f)
	c.Assert(err, IsNil)
	c.Assert(result, Equals, string(fc))
}
