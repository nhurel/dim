package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Config holds server configuration
type Config struct {
	Port           string
	Authorizations []*Authorization
}

// Authorization defines restrictions to call a given URL
type Authorization struct {
	Path       string
	Method     string
	Users      []*Credentials
	pathRegexp *regexp.Regexp
}

// Credentials define a user credentials who can be granted authorizations
type Credentials struct {
	Username, Password string
}

// CompilePath compiles this Authorization Path member as a regexp
func (auth *Authorization) CompilePath() error {
	var err error
	path := auth.Path
	if !strings.HasSuffix(path, "$") {
		path += ".*"
	}
	if auth.pathRegexp, err = regexp.Compile(path); err != nil {
		return fmt.Errorf("Failed to parse path %s : %v", auth.Path, err)
	}
	return nil
}

// Applies indicates this Authorization matches the given request
func (auth *Authorization) Applies(req *http.Request) bool {
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	return auth.pathRegexp.MatchString(path) && (auth.Method == "" || auth.Method == req.Method)
}
