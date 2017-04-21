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
	"io/ioutil"
	"sync"
	"text/template"

	"github.com/Sirupsen/logrus"
	"github.com/nhurel/dim/lib"
)

// Config holds index configuration
type Config struct {
	// Directory where to write index data
	Directory string
	// Hooks to trigger on event
	Hooks   []*Hook
	funcMap template.FuncMap
}

// Hook evals the template string  when an event of its type occurs
type Hook struct {
	Event  dim.ActionType
	Action string
	eval   *template.Template
}

// RegisterFunction adds a function that can be used in hook Actions
func (c *Config) RegisterFunction(name string, function interface{}) {
	if c.funcMap == nil {
		c.funcMap = make(map[string]interface{})
	}
	c.funcMap[name] = function
}

// ParseHooks parses Action fields of all hooks and stores the template in their eval fields
// Functions available in templates should be added before with RegisterFunction
func (c *Config) ParseHooks() error {
	var err error
	for i, h := range c.Hooks {
		logrus.WithField("hook", h).Debugln("Parsing hook")
		name := fmt.Sprintf("hook_%d", i+1)
		if h.Event != dim.PushAction && h.Event != dim.DeleteAction {
			return fmt.Errorf("Unknown event %s. Only %s and %s supported", h.Event, dim.PushAction, dim.DeleteAction)
		}

		var tpl *template.Template
		if tpl, err = template.New(name).Funcs(c.funcMap).Parse(h.Action); err != nil {
			return fmt.Errorf("Failed to parse %s : %v", name, err)
		}
		//c.Hooks[i].eval = tpl
		h.eval = tpl
	}
	return nil
}

// GetHooks return all hooks for a given ActionType
func (c *Config) GetHooks(event dim.ActionType) []*Hook {
	hooks := make([]*Hook, 0, len(c.Hooks))
	for _, h := range c.Hooks {
		if h.Event == event {
			hooks = append(hooks, h)
		}
	}
	return hooks
}

var mutex = &sync.Mutex{}

// Eval runs the template with the given image as parameter
func (h *Hook) Eval(image *dim.IndexImage) error {
	if h.eval == nil {
		return fmt.Errorf("Cannot eval hook, it has no template : %v", h)
	}
	mutex.Lock()
	defer mutex.Unlock()
	return h.eval.Execute(ioutil.Discard, image)
}
