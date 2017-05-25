package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Sirupsen/logrus"
)

// RegistryProxy controls access to registry endpoints and forwards request when user is granted
type RegistryProxy struct {
	target             *url.URL
	username, password string
	proxy              *httputil.ReverseProxy
}

// NewRegistryProxy creates a RegistryProxy instance
func NewRegistryProxy(registryURL *url.URL, username, password string) *RegistryProxy {
	// TODO inject an object that reads user from request (basic auth or other)
	rp := &RegistryProxy{target: registryURL, username: username, password: password}
	rp.proxy = httputil.NewSingleHostReverseProxy(registryURL)
	return rp
}

// Forwards sends request to the actual docker registry
func (rp *RegistryProxy) Forwards(w http.ResponseWriter, r *http.Request) {
	// TODO implement access controls
	if false {
		w.WriteHeader(http.StatusForbidden)
	}

	logrus.WithFields(logrus.Fields{"registryURL": rp.target, "targetURL": r.RequestURI}).Infoln("Forwarding request to target registry")

	if rp.username != "" && rp.password != "" {
		r.SetBasicAuth(rp.username, rp.password)
	}

	if r.TLS != nil {
		r.Header.Set("X-Forwarded-Proto", "https")
	}
	r.Header.Set("X-Forwarded-Host", r.Host)

	if rp.target.Scheme == "https" {
		r.URL.Scheme = "https"
	}
	if rp.target.Host != r.Host {
		r.URL.Host = rp.target.Host
		r.Host = rp.target.Host
	}

	rp.proxy.ServeHTTP(w, r)
}
