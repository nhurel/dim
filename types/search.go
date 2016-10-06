package types

import "time"

// SearchResult describes a search result returned from a registry
type SearchResult struct {
	// StarCount indicates the number of stars this repository has (not supported in private registry)
	StarCount int `json:"star_count"`
	// IsOfficial is true if the result is from an official repository. (not supported in private registry)
	IsOfficial bool `json:"is_official"`
	// Name is the name of the repository
	Name string `json:"name"`
	// IsAutomated indicates whether the result is automated (not supported in private registry)
	IsAutomated bool `json:"is_automated"`
	// Description is a textual description of the repository (filled with the tag of the repo)
	Description string `json:"description"`
	// Tag identifie one version of the image
	Tag string `json:"tag"`
	// FullName stores the fully qualified name of the image
	FullName string `json:"full_name"`
	// Created is the time when the image was created
	Created time.Time `json:"created"`
	// Label is an array holding all the labels applied to  an image
	Label map[string]string `json:"label"`
	// Volumes is an array holding all volumes declared by the image
	Volumes []string `json:"volumes"`
	// Exposed port is an array containing all the ports exposed by an image
	ExposedPorts []int `json:"exposed_ports"`
	// Env is a map of all environment variables
	Env map[string]string `json:"env"`
	// Size is the size of the image
	Size int64 `json:"size"`
}

// SearchResults lists a collection search results returned from a registry
type SearchResults struct {
	// Query contains the query string that generated the search results
	Query string `json:"query"`
	// NumResults indicates the number of results the query returned
	NumResults int `json:"num_results"`
	// Results is a slice containing the actual results for the search
	Results []SearchResult `json:"results"`
}
