package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"github.com/Sirupsen/logrus"
)

// RegistryProxy controls access to registry endpoints and forwards request when user is granted
type RegistryProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

// NewRegistryProxy creates a RegistryProxy instance
func NewRegistryProxy(registryURL *url.URL) *RegistryProxy {
	// TODO inject an object that reads user from request (basic auth or other)
	rp := &RegistryProxy{target: registryURL}
	rp.proxy = httputil.NewSingleHostReverseProxy(registryURL)
	return rp
}

// Forwards sends request to the actual docker registry
func (rp *RegistryProxy) Forwards(w http.ResponseWriter, r *http.Request) {
	// TODO implement access controls
	if false {
		w.WriteHeader(http.StatusForbidden)
	}
	logrus.WithFields(logrus.Fields{"registryURL": rp.target, "targetURL": r.URL}).Infoln("Forwarding request to target registry")

	rp.proxy.ServeHTTP(w, r)
}
