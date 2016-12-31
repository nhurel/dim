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

package index

import (
	"fmt"
	"testing"

	"github.com/nhurel/dim/lib"
)

func TestParseHooks(t *testing.T) {
	scenarii := []struct {
		Event    dim.ActionType
		Actions  []string
		expected error
	}{
		{"", nil, nil},
		{"", []string{"wrong"}, fmt.Errorf("Unknown event . Only push and delete supported")},
		{dim.DeleteAction, []string{"wrong{{}}"}, fmt.Errorf("Failed to parse hook_1 : template: hook_1:1: missing value for command")},
		{dim.PushAction, []string{"coorect"}, nil},
		{dim.DeleteAction, []string{"correct", "wrong{{}}}"}, fmt.Errorf("Failed to parse hook_2 : template: hook_2:1: missing value for command")},
	}

	for _, scenario := range scenarii {
		hooks := make([]*Hook, len(scenario.Actions))
		for i, a := range scenario.Actions {
			hooks[i] = &Hook{Event: scenario.Event, Action: a}
		}
		c := &Config{Hooks: hooks}
		got := c.ParseHooks()

		if (got == nil) != (scenario.expected == nil) {
			t.Errorf("ParseHooks(%v) returned %v instead of %v", scenario, got, scenario.expected)
		} else if got != nil && got.Error() != scenario.expected.Error() {
			t.Errorf("ParseHooks(%v) returned %v instead of %v", scenario, got, scenario.expected)
		}
	}
}

func TestGetHooks(t *testing.T) {
	deleteHook := &Hook{Event: dim.DeleteAction}
	pushHook := &Hook{Event: dim.PushAction}
	scenarii := []struct {
		given     []*Hook
		requested dim.ActionType
		expected  []*Hook
	}{
		{
			given:     []*Hook{deleteHook, pushHook},
			requested: dim.PushAction,
			expected:  []*Hook{pushHook},
		},
		{
			given:     []*Hook{deleteHook, pushHook},
			requested: dim.DeleteAction,
			expected:  []*Hook{deleteHook},
		},
		{
			given:     []*Hook{pushHook},
			requested: dim.DeleteAction,
			expected:  []*Hook{},
		},
	}

	for _, scenario := range scenarii {
		c := &Config{Hooks: scenario.given}
		got := c.GetHooks(scenario.requested)
		if !hookEquals(got, scenario.expected) {
			t.Errorf("GetHooks(%s) on %v returned %v instead of %v", scenario.requested, scenario.given, got, scenario.expected)
		}
	}
}

func TestRegisterFunction(t *testing.T) {
	c := &Config{}
	c.RegisterFunction("log", func() error { return nil })
	if len(c.funcMap) != 1 {
		t.Fatalf("config should have 1 function but there is %d : %v", len(c.funcMap), c.funcMap)
	}
	if c.funcMap["log"] == nil {
		t.Fatalf("config should have log function but is has %v", c.funcMap)
	}
}

func hookEquals(h1, h2 []*Hook) bool {
	if len(h1) != len(h2) {
		return false
	}
	for i, h := range h1 {
		if h != h2[i] {
			return false
		}
	}
	return true
}
