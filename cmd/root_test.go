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

package cmd

import (
	"fmt"
	"testing"

	"github.com/Sirupsen/logrus"
)

func TestGuessTag(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	tests := []struct {
		tagOption     string
		imageName     string
		imageTags     []string
		override      bool
		exepetedTag   string
		expectedError error
	}{
		{"imagename:newtag", "imagename:tag", []string{"imagename:tag"}, false, "imagename:newtag", nil},
		{"", "imagename:tag", []string{"imagename:tag"}, true, "imagename:tag", nil},
		{"", "imageID", []string{}, true, "", fmt.Errorf("Cannot override image with no tag. Use --tag option instead")},
		{"", "imageID", []string{"imagename:tag"}, true, "imagename:tag", nil},
		{"imagename:tag", "imagename:tag", []string{"imagename:tag"}, false, "imagename:tag", nil},
	}

	for _, test := range tests {

		tag, err := guessTag(test.tagOption, test.imageName, test.imageTags, test.override)

		if tag != test.exepetedTag {
			t.Errorf("GuessTag returned wrong tag. Expected %s - Got %s", test.exepetedTag, tag)
		}

		if (err == nil && test.expectedError != nil) || (err != nil && test.expectedError == nil) || (err != nil && err.Error() != test.expectedError.Error()) {
			t.Errorf("GuessTag did'nt return exepectede error. Expected %v - Got %v", err, test.expectedError)
		}

	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		image       string
		registryURL string
		expected    string
	}{
		{"imagename", "https://private-registry.com", "private-registry.com/imagename"},
		{"imagename:latest", "https://private-registry.com", "private-registry.com/imagename:latest"},
		{"imagename:tag", "http://private-registry.com", "private-registry.com/imagename:tag"},
		{"imagename:tag", "http://private-registry.com:5000", "private-registry.com:5000/imagename:tag"},
	}
	for _, test := range tests {

		got, err := parseName(test.image, test.registryURL)

		if err != nil {
			t.Fatalf("pareseNamed return error %v", err)
		}

		if got.String() != test.expected {
			t.Errorf("parseName returned %s. Expected %s", got.String(), test.expected)
		}

	}

}
