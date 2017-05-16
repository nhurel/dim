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
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/nhurel/dim/lib/utils"
)

// GetAuthorization finds the first Authorization matching the request
func GetAuthorization(req *http.Request, auths []*Authorization) *Authorization {
	for _, auth := range auths {
		if auth.Applies(req) {
			return auth
		}
	}
	return nil
}

const authenticateHeaderName = "WWW-Authenticate"
const authenticateHeaderValue = "Basic realm=\"Registry Authentication\""

func securityFilter(cfg *Config, hf http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := GetAuthorization(r, cfg.Authorizations)
		if auth != nil {
			if err := grantAccess(r, auth); err != nil {
				u, _, _ := r.BasicAuth()
				logrus.WithFields(logrus.Fields{"username": u, "url": r.URL}).Infoln("Rejecting request")
				w.Header().Set(authenticateHeaderName, authenticateHeaderValue)
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
		}
		hf(w, r)
	}
}

func grantAccess(req *http.Request, auth *Authorization) error {
	if auth.Users != nil {
		for _, user := range auth.Users {
			u, p, ok := req.BasicAuth()
			if ok {
				if u == user.Username && utils.Sha256(p) == user.Password {
					return nil
				}
			}
		}
		return fmt.Errorf("You are not allowed to access URLs matching %s", auth.Path)
	}
	return nil
}
