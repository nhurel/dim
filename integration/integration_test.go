package integration

import (
	"github.com/docker/engine-api/types"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/wrapper/dockerClient"
	. "gopkg.in/check.v1"
	"os/exec"
	"regexp"
	"testing"
	"time"
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
}

var integration_label = testImages{
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

func (s *IntegrationTestSuite) SetUpSuite(c *C) {
	s.Dim = &dim.Dim{Docker: &dockerClient.DockerClient{}}
	if err := s.Dim.Pull("httpd:2.4-alpine"); err != nil {
		c.Error(err)
	}
}

func (s *IntegrationTestSuite) TestLabelAndSearch(c *C) {
	if err := s.Dim.AddLabel("httpd:2.4-alpine", integration_label.labels, integration_label.tag); err != nil {
		c.Error(err)
	}
	if err := s.Dim.Push(integration_label.tag, &types.AuthConfig{}); err != nil {
		c.Error(err)
	}
	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	queries := []string{"Label.type:web", "Label.first:true"}

	for _, q := range queries {
		c.Logf("Checking query %s returns httpd:first\n", q)
		result, err := exec.Command(dimExec, "search", "--registry-url=localhost", "-k", "-a", q, "-l debug").Output()
		c.Log(string(result))
		if err != nil {
			c.Log(string(err.(*exec.ExitError).Stderr))
		}
		c.Assert(err, IsNil)
		re := regexp.MustCompile("httpd\\s*first")
		c.Assert(re.MatchString(string(result)), Equals, true)
	}

}

func (s *IntegrationTestSuite) TestUnlabelAndSearch(c *C) {
	if err := s.Dim.AddLabel("httpd:2.4-alpine", integration_label.labels, integration_label.tag); err != nil {
		c.Error(err)
	}
	if err := s.Dim.RemoveLabel(integration_label.tag, integration_label.labelsName, integration_label.tag); err != nil {
		c.Error(err)
	}
	if err := s.Dim.Push(integration_label.tag, &types.AuthConfig{}); err != nil {
		c.Error(err)
	}
	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	queries := []string{"Label.type:web", "Label.first:true", "Labels.type:/*/"}
	for _, q := range queries {
		c.Logf("Checking query %s returns 0 result\n", q)
		result, err := exec.Command(dimExec, "search", "--registry-url=localhost", "-k", "-a", q, "-l debug").Output()
		c.Log(string(result))
		if err != nil {
			c.Log(string(err.(*exec.ExitError).Stderr))
		}
		c.Assert(err, IsNil)
		re := regexp.MustCompile("No result found")
		//c.Assert(string(result), Matches, "\n.*0 Results found.*")
		c.Assert(re.MatchString(string(result)), Equals, true)
	}
}

func (s *IntegrationTestSuite) TestDeleteAndSearch(c *C) {
	if err := s.Dim.AddLabel("httpd:2.4-alpine", integration_label.labels, integration_label.tag); err != nil {
		c.Error(err)
	}
	if err := s.Dim.Push(integration_label.tag, &types.AuthConfig{}); err != nil {
		c.Error(err)
	}
	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image
	if _, err := exec.Command(dimExec, "delete", "--registry-url=localhost", "-k", "-r", integration_label.tag, "-l debug").Output(); err != nil {
		c.Log(string(err.(*exec.ExitError).Stderr))
		c.Error("Error when deleting image")
	}
	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	result, err := exec.Command(dimExec, "search", "--registry-url=localhost", "-k", "httpd", "-l debug").Output()
	c.Log(string(result))
	if err != nil {
		c.Log(string(err.(*exec.ExitError).Stderr))
	}
	c.Assert(err, IsNil)
	re := regexp.MustCompile("No result found")
	//c.Assert(string(result), Matches, "\n.*0 Results found.*")
	c.Assert(re.MatchString(string(result)), Equals, true)

}
