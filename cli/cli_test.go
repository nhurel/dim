package cli

import (
	"bytes"
	"testing"

	"github.com/docker/docker/api/types"
)

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
