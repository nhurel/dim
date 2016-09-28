package utils

import (
	"testing"
)

func TestBuildURL(t *testing.T) {

	//GIVEN

	scenarii := []struct {
		baseURL  string
		insecure bool
		expected string
	}{
		{"google.fr", false, "https://google.fr"},
		{"http://google.fr", false, "http://google.fr"},
		{"google.fr", true, "http://google.fr"},
		{"https://google.fr", true, "https://google.fr"},
	}

	for _, scenario := range scenarii {
		// WHEN
		got := BuildURL(scenario.baseURL, scenario.insecure)
		// THEN
		if got != scenario.expected {
			t.Errorf("BuildURL returned %s. Expected : %s", got, scenario.expected)
		}
	}
}
