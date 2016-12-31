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

package dim_test

import (
	"fmt"
	"testing"

	"bytes"

	"strings"

	"github.com/docker/docker/utils/templates"
	"github.com/nhurel/dim/lib"
	"github.com/nhurel/dim/lib/mock"
	"github.com/nhurel/dim/lib/utils"
)

func TestAddLabel(t *testing.T) {
	// GIVEN

	parent := "parentImage:test"
	tests := []struct {
		labels             []string
		mockGetImageLabels map[string]string
		expected           error
	}{
		{labels: []string{}, expected: fmt.Errorf("No label provided")},
		{labels: []string{""}, expected: fmt.Errorf("Failed to parse given label ")},
		{labels: []string{"label1=value1"}},
		{labels: []string{"label1NoValue"}, expected: fmt.Errorf("Failed to parse given label label1NoValue")},
		{labels: []string{"label1=value1", "label2=value2"}},
		{
			labels:             []string{"label1=value1", "label2=value2"},
			mockGetImageLabels: map[string]string{"label1": "value1", "label2": "value2"},
			expected:           fmt.Errorf("Image parentImage:test already contains the label(s) you want to set"),
		},
		{labels: []string{"label1=value1", "label2NoValue"}, expected: fmt.Errorf("Failed to parse given label label2NoValue")},
	}

	for _, test := range tests {
		d := &dim.Dim{Docker: &mock.NoOpDockerClient{ImageInspectLabels: test.mockGetImageLabels, Calls: make(map[string][]interface{})}}
		//WHEN
		err := d.AddLabel(parent, test.labels, "")

		//THEN
		if (err == nil && test.expected != nil) || (err != nil && test.expected == nil) || (err != nil && err.Error() != test.expected.Error()) {
			t.Errorf("Wrong error returned when label is %v. Got %v - Expected %v", test.labels, err, test.expected)
		}
	}
}

func TestRemoveLabel(t *testing.T) {
	parent := "parentImage:test"
	tag := "childImage:latest"
	tests := []struct {
		labels             []string
		mockGetImageLabels map[string]string
		expectedError      error
		expectedLabelsCall []interface{}
	}{
		{labels: []string{}, mockGetImageLabels: nil, expectedError: fmt.Errorf("No label provided"), expectedLabelsCall: nil},
		{
			labels:             []string{"unknown", "known"},
			mockGetImageLabels: map[string]string{"known": "whatever"},
			expectedError:      nil,
			expectedLabelsCall: []interface{}{parent, map[string]string{"known": ""}, tag},
		},
		{
			labels:             []string{"unknown"},
			mockGetImageLabels: map[string]string{"known": "whatever"},
			expectedError:      fmt.Errorf("Image parentImage:test has none of the given label(s) you want to clear"),
			expectedLabelsCall: nil,
		},
		{
			labels:             []string{"known=whatever"},
			mockGetImageLabels: map[string]string{"known": "whatever"},
			expectedError:      fmt.Errorf("Failed to parse given label known=whatever"),
			expectedLabelsCall: nil,
		},
	}

	for _, test := range tests {
		calls := make(map[string][]interface{})
		d := &dim.Dim{Docker: &mock.NoOpDockerClient{ImageInspectLabels: test.mockGetImageLabels, Calls: calls}}
		got := d.RemoveLabel(parent, test.labels, tag)

		if test.expectedError != nil {
			if got == nil || got.Error() != test.expectedError.Error() {
				t.Errorf("RemoveLabel returned the wrong error. Expected %v - Got %v", test.expectedError, got)
			}
		} else {
			if got != nil {
				t.Errorf("RemoveLabel returned error %v. No error was expected", got)
			}
			if test.expectedLabelsCall != nil {
				testNoOpCalls(calls, "ImageBuild", test.expectedLabelsCall, t)
			}
		}

	}

}

func TestRemove(t *testing.T) {
	//GIVEN
	calls := make(map[string][]interface{})
	d := &dim.Dim{Docker: &mock.NoOpDockerClient{Calls: calls}}
	//WHEN
	d.Remove("image")
	//THEN
	testNoOpCalls(calls, "Remove", []interface{}{"image"}, t)

}

func TestPush(t *testing.T) {
	//GIVEN
	calls := make(map[string][]interface{})
	d := &dim.Dim{Docker: &mock.NoOpDockerClient{Calls: calls}}
	//WHEN
	d.Push("image")
	//THEN
	testNoOpCalls(calls, "Push", []interface{}{"image"}, t)

}

func TestPull(t *testing.T) {
	//GIVEN
	calls := make(map[string][]interface{})
	d := &dim.Dim{Docker: &mock.NoOpDockerClient{Calls: calls}}
	//WHEN
	d.Pull("image")
	//THEN
	testNoOpCalls(calls, "Pull", []interface{}{"image"}, t)
}

func TestGetImageInfo(t *testing.T) {
	//GIVEN
	calls := make(map[string][]interface{})
	d := &dim.Dim{Docker: &mock.NoOpDockerClient{Calls: calls}}
	//WHEN
	d.GetImageInfo("image")
	//THEN
	testNoOpCalls(calls, "Inspect", []interface{}{"image"}, t)
}

func TestPrintImageInfo(t *testing.T) {
	//GIVEN
	d := &dim.Dim{Docker: &mock.NoOpDockerClient{ImageInspectLabels: map[string]string{"key1": "value1"}, Calls: make(map[string][]interface{})}}
	b := make([]byte, 1000)
	writer := bytes.NewBuffer(b)
	//WHEN
	tpl, err := templates.NewParse("test", "Labels:{{range $k, $v := .Config.Labels}}{{$k}} = {{$v}}{{end}}")
	if err != nil {
		t.Fatal(err)
	}
	d.PrintImageInfo(writer, "image", tpl)
	got := strings.TrimSpace(writer.String())
	//THEN
	if !strings.Contains(got, "Labels:key1 = value1") {
		t.Errorf("PrintImageInfo returned '%s'", got)
	}

}

func testNoOpCalls(calls map[string][]interface{}, method string, expectedParams []interface{}, t *testing.T) {
	if len(calls[method]) != len(expectedParams) {
		t.Errorf("%s was called with %d parameters. Expected %d", method, len(calls[method]), len(expectedParams))
	}
	for ind, param := range calls[method] {
		switch p := param.(type) {
		case map[string]string:
			if utils.FlatMap(p) != utils.FlatMap(expectedParams[ind].(map[string]string)) {
				t.Errorf("%s was called with parameter #%d : %v. Expected %v", method, ind, param, expectedParams)
			}
		default:
			if p != expectedParams[ind] {
				t.Errorf("%s was called with parameter #%d : %v. Expected %v", method, ind, param, expectedParams)
			}
		}
	}
}
