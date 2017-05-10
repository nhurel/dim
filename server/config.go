package server

// Config holds server configuration
type Config struct {
	Port           string
	Authorizations []*Authorization
}

// Authorization defines restrictions to call a given URL
type Authorization struct {
	Path     string
	Method   string
	Username string
	Password string
}
