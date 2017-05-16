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

package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nhurel/dim/lib/utils"
)

func TestGrantAccess(t *testing.T) {
	tests := []struct {
		givenUsername, givenPassword string
		auth                         *Authorization
		expected                     error
	}{
		{auth: &Authorization{Users: []*Credentials{{Username: "private", Password: utils.Sha256("secure")}}, Path: "/forbdden/path"}, expected: errors.New("You are not allowed to access URLs matching /forbdden/path")},
		{givenUsername: "private", givenPassword: "secure", auth: &Authorization{Users: []*Credentials{{Username: "private", Password: utils.Sha256("secure")}}, Path: "/forbdden/path"}},
		{givenUsername: "forbidden", givenPassword: "badPassword", auth: &Authorization{Users: []*Credentials{{Username: "private", Password: utils.Sha256("secure")}}, Path: "/forbdden/path"}, expected: errors.New("You are not allowed to access URLs matching /forbdden/path")},
		{auth: &Authorization{Path: "/forbdden/path"}, expected: nil},
		{givenUsername: "private", givenPassword: "secure", auth: &Authorization{Path: "/forbdden/path"}, expected: nil},
	}

	for i, test := range tests {
		req := httptest.NewRequest(http.MethodGet, test.auth.Path, nil)
		if test.givenUsername != "" {
			req.SetBasicAuth(test.givenUsername, test.givenPassword)
		}
		err := grantAccess(req, test.auth)
		if (err == nil) != (test.expected == nil) {
			t.Errorf("Test #%d grantAccess returned : %v but it was expected : %v", i, err, test.expected)
			continue
		}
		if err != nil && err.Error() != test.expected.Error() {
			t.Errorf("Test #%d grantAccess returned : %v but it was expected : %v", i, err, test.expected)
		}
	}
}

func TestGetAuthorization(t *testing.T) {
	auths := []*Authorization{
		{Path: "/prefix/$"},
		{Path: "/public"},
		{Path: "/secure", Users: []*Credentials{{Username: "user", Password: "p"}}},
		{Path: "/method", Method: http.MethodGet},
		{Path: "/method", Method: http.MethodPost},
		{Path: "/prefix"},
		{Path: "/"},
	}

	for _, auth := range auths {
		auth.CompilePath()
	}
	tests := []struct {
		givenURL    string
		givenMethod string
		expected    *Authorization
	}{
		{givenURL: "http://example.com", givenMethod: http.MethodGet, expected: auths[6]},
		{givenURL: "/", givenMethod: http.MethodGet, expected: auths[6]},
		{givenURL: "/unknown", givenMethod: http.MethodGet, expected: auths[6]},
		{givenURL: "/public", givenMethod: http.MethodGet, expected: auths[1]},
		{givenURL: "/public/", givenMethod: http.MethodGet, expected: auths[1]},
		{givenURL: "/public/subpath", givenMethod: http.MethodGet, expected: auths[1]},
		{givenURL: "/secure", givenMethod: http.MethodGet, expected: auths[2]},
		{givenURL: "/method", givenMethod: http.MethodGet, expected: auths[3]},
		{givenURL: "/method", givenMethod: http.MethodPost, expected: auths[4]},
		{givenURL: "/prefix/", givenMethod: http.MethodGet, expected: auths[0]},
		{givenURL: "/prefix", givenMethod: http.MethodGet, expected: auths[5]},
		{givenURL: "/prefix/subpath", givenMethod: http.MethodGet, expected: auths[5]},
	}

	for i, test := range tests {
		r := httptest.NewRequest(test.givenMethod, test.givenURL, nil)
		got := GetAuthorization(r, auths)
		if got != test.expected {
			t.Errorf("GetAuthorization#%d returned %+v but it was expected %+v", i, got, test.expected)
		}
	}

}
