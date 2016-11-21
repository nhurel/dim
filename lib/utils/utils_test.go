package utils

import (
	"bytes"
	"sort"
	"testing"

	"time"

	"github.com/docker/engine-api/types"
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

func TestListContains(t *testing.T) {
	scenarii := []struct {
		list     []string
		search   string
		expected bool
	}{
		{
			list:     []string{"one", "two", "three", "four", "five"},
			search:   "one",
			expected: true,
		},
		{
			list:     []string{"one", "two", "three", "four", "five"},
			search:   "four",
			expected: true,
		},
		{
			list:     []string{"one", "two", "three", "four", "five"},
			search:   "cinq",
			expected: false,
		},
	}

	for _, scenario := range scenarii {
		// WHEN
		got := ListContains(scenario.list, scenario.search)
		// THEN
		if got != scenario.expected {
			t.Errorf("ListContains(%v, %s) returned %t instead of %t ", scenario.list, scenario.search, got, scenario.expected)
		}
	}
}

func TestMapMatchesAll(t *testing.T) {
	scenarii := []struct {
		first, second map[string]string
		expected      bool
	}{
		{
			first:    map[string]string{"one": "1", "two": "2", "three": "3", "four": "4", "five": "5"},
			second:   map[string]string{"one": "1", "two": "2", "three": "3", "four": "4", "five": "5"},
			expected: true,
		},
		{
			first:    map[string]string{"one": "1", "two": "2", "three": "3", "four": "4", "five": "5"},
			second:   map[string]string{"one": "1", "two": "2", "three": "3"},
			expected: true,
		},
		{
			first:    map[string]string{"one": "1", "two": "2", "three": "3"},
			second:   map[string]string{"one": "1", "two": "2", "three": "3", "four": "4", "five": "5"},
			expected: false,
		},
		{
			first:    map[string]string{"one": "1", "two": "2", "three": "3"},
			second:   map[string]string{"one": "3", "two": "4", "three": "5"},
			expected: false,
		},
		{
			first:    nil,
			second:   map[string]string{"one": "1", "two": "2", "three": "3"},
			expected: false,
		},
		{
			first:    map[string]string{"one": "1", "two": "2", "three": "3"},
			second:   nil,
			expected: false,
		},
	}

	for _, scenario := range scenarii {
		// WHEN
		got := MapMatchesAll(scenario.first, scenario.second)
		// THEN
		if got != scenario.expected {
			t.Errorf("MapMatchesAll(%v, %v) returned %t instead of %t ", scenario.first, scenario.second, got, scenario.expected)
		}
	}
}

func TestKeys(t *testing.T) {
	scenarii := []struct {
		given    map[string]string
		expected []string
	}{
		{
			given:    map[string]string{"one": "1", "two": "2", "three": "3"},
			expected: []string{"one", "three", "two"},
		},
		{
			given:    map[string]string{},
			expected: []string{},
		},
		{
			given:    nil,
			expected: []string{},
		},
	}

	for _, scenario := range scenarii {
		//WHEN
		got := Keys(scenario.given)
		if len(got) != len(scenario.expected) {
			t.Errorf("Keys returned only %d elements %v. Expected to get %d elements from %v keys", len(got), got, len(scenario.expected), scenario.given)
		} else if scenario.expected != nil {
			sort.Strings(got)
			for i, e := range scenario.expected {
				if got[i] != e {
					t.Errorf("Keys did not return %s from %v. Got %v expected %v", e, scenario.given, got, scenario.expected)
				}
			}
		}
	}
}

func TestFilterValues(t *testing.T) {
	scenarii := []struct {
		given    map[string]string
		filter   string
		expected map[string]string
	}{
		{
			given:    map[string]string{"one": "1", "two": "2", "three": "3"},
			filter:   "2",
			expected: map[string]string{"one": "1", "three": "3"},
		},
		{
			given:    map[string]string{"one": "1", "two": "", "three": "3"},
			filter:   "",
			expected: map[string]string{"one": "1", "three": "3"},
		},
		{
			given:    nil,
			filter:   "",
			expected: map[string]string{},
		},
		{
			given:    map[string]string{},
			filter:   "",
			expected: map[string]string{},
		},
		{
			given:    map[string]string{"one": "1", "two": "2", "three": "3"},
			filter:   "",
			expected: map[string]string{"one": "1", "two": "2", "three": "3"},
		},
	}

	for _, scenario := range scenarii {
		//WHEN
		got := FilterValues(scenario.given, scenario.filter)
		//THEN
		if scenario.expected == nil {
			if got != nil {
				t.Errorf("FilterValues(%v, %s) returned %v instead of nil", scenario.given, scenario.filter, got)
			}
		} else {
			if len(got) != len(scenario.expected) {
				t.Errorf("FilterValues(%v, %s) returned %v instead of %v", scenario.given, scenario.filter, got, scenario.expected)
			} else {
				for ek, ev := range scenario.expected {
					if got[ek] != ev {
						t.Errorf("FilterValues(%v, %s) returned %v instead of %v. %s key don't match", scenario.given, scenario.filter, got, scenario.expected, ek)
					}
				}
			}
		}

	}

}

func TestFlatMap(t *testing.T) {
	scenarii := []struct {
		given    map[string]string
		expected string
	}{
		{
			given:    map[string]string{"one": "1", "two": "2", "three": "3"},
			expected: "one=1, three=3, two=2",
		},
		{
			given:    map[string]string{"one": "1"},
			expected: "one=1",
		},
		{
			given:    nil,
			expected: "",
		},
		{
			given:    map[string]string{},
			expected: "",
		},
	}

	for _, scenario := range scenarii {
		//WHEN
		got := FlatMap(scenario.given)
		if got != scenario.expected {
			t.Errorf("FlatMap(%v) returned '%s' instead of '%s'", scenario.given, got, scenario.expected)
		}
	}

}

func mockRead(s string) InputReader {
	return func(a ...interface{}) (int, error) {
		if a, ok := a[0].(*string); ok {
			*a = s
		}
		return 1, nil
	}
}

func TestAskUsername(t *testing.T) {
	scenarii := []struct {
		username         string
		mockedInput      string
		expectedUsername string
		expectedPrompt   string
	}{
		{username: "", mockedInput: "", expectedUsername: "", expectedPrompt: "Username :"},
		{username: "", mockedInput: "login", expectedUsername: "login", expectedPrompt: "Username :"},
		{username: "rememberme", mockedInput: "", expectedUsername: "rememberme", expectedPrompt: "Username (rememberme) :"},
		{username: "rememberme", mockedInput: "login", expectedUsername: "login", expectedPrompt: "Username (rememberme) :"},
	}

	for _, scenario := range scenarii {
		sw := bytes.NewBuffer([]byte{})
		auth := &types.AuthConfig{Username: scenario.username}
		askUsername(auth, mockRead(scenario.mockedInput), sw)
		if auth.Username != scenario.expectedUsername {
			t.Errorf("askUsername(%s) returned '%s' instead of '%s'", scenario.username, auth.Username, scenario.expectedUsername)
		}
		if sw.String() != scenario.expectedPrompt {
			t.Errorf("askUsername(%s) used prompt'%s' instead of '%s'", scenario.username, sw.String(), scenario.expectedPrompt)
		}

	}
}

func TestAskPassword(t *testing.T) {
	scenarii := []struct {
		password         string
		mockedInput      string
		expectedPassword string
	}{
		{password: "", mockedInput: "", expectedPassword: ""},
		{password: "remember", mockedInput: "", expectedPassword: "remember"},
		{password: "", mockedInput: "password", expectedPassword: "password"},
		{password: "remember", mockedInput: "password", expectedPassword: "password"},
	}

	for _, scenario := range scenarii {
		sw := bytes.NewBuffer([]byte{})
		auth := &types.AuthConfig{Password: scenario.password}
		askPassword(auth, mockRead(scenario.mockedInput), sw)
		if auth.Password != scenario.expectedPassword {
			t.Errorf("askPassword(%s) returned '%s' instead of '%s'", scenario.password, auth.Password, scenario.expectedPassword)
		}
	}
}

func TestParseDuration(t *testing.T) {
	scenarii := []struct {
		since    time.Duration
		expected string
	}{
		{time.Hour + time.Minute, "1 hour ago"},
		{3*time.Hour + 10*time.Minute, "3 hours ago"},
		{time.Minute + 50*time.Second, "2 minutes ago"},
		{time.Minute + 10*time.Second, "1 minute ago"},
		{15*time.Minute + 29*time.Second, "15 minutes ago"},
		{30 * time.Second, "30 seconds ago"},
		{1 * time.Second, "1 second ago"},
		{2 * time.Hour, "2 hours ago"},
		{30 * time.Hour, "30 hours ago"},
		{48 * time.Hour, "2 days ago"},
		{24 * 7 * time.Hour, "7 days ago"},
		{24 * 12 * time.Hour, "12 days ago"},
		{24 * 7 * 2 * time.Hour, "2 weeks ago"},
		{24 * 7 * 6 * time.Hour, "6 weeks ago"},
		{24 * 7 * 8 * time.Hour, "2 months ago"},
		{24 * 7 * 4 * 18 * time.Hour, "18 months ago"},
	}

	for _, scenario := range scenarii {
		if got := ParseDuration(scenario.since); got != scenario.expected {
			t.Errorf("ParseDuration(%s) returned '%s' instead of '%s'", scenario.since.String(), got, scenario.expected)
		}

	}
}
