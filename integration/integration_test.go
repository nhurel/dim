package integration

import (
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

	o, err := exec.Command(command, params...).Output()
	return string(o), err

}

func (s *IntegrationTestSuite) SetUpSuite(c *C) {
	s.Dim = &dim.Dim{Docker: &dockerClient.DockerClient{Insecure: true}}
	if err := s.Dim.Pull("httpd:2.4-alpine"); err != nil {
		c.Error(err)
	}
}

func (s *IntegrationTestSuite) TestLabelAndSearch(c *C) {
	if o, err := runCommand(dimExec, "label", "httpd:2.4-alpine", "-t", integration_label.tag, "-p", "-k", "-r", integration_label.labels); err != nil {
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
		re := regexp.MustCompile("httpd\\s*first")
		c.Assert(re.MatchString(string(result)), Equals, true)
	}

}

func (s *IntegrationTestSuite) TestUnlabelAndSearch(c *C) {
	if o, err := runCommand(dimExec, "label", "httpd:2.4-alpine", "-t", integration_label.tag, "-p", integration_label.labels); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	if o, err := runCommand(dimExec, "label", "-d", "-o", "-r", integration_label.tag, "-p", "-k", integration_label.labelsName); err != nil {
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
	if o, err := runCommand(dimExec, "label", "-p", "httpd:2.4-alpine", "-t", integration_label.tag, "-r", "-k", integration_label.labels); err != nil {
		c.Error(o)
		c.Fatal(err)
	}

	time.Sleep(750 * time.Millisecond) // tempo to make sure dim indexes the image

	if o, err := runCommand(dimExec, "delete", "--registry-url=localhost", "-k", "-r", integration_label.tag); err != nil {
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
