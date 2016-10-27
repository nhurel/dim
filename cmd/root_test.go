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
