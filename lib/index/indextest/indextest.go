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

package indextest

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
)

// ParseTime converts a given string into time
func ParseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		logrus.WithError(err).WithField("time", value).Error("failed to parse datetime")
	}
	return t
}

// MockIndex creates a bleve Index for tests
func MockIndex(dm *bleve.DocumentMapping) (bleve.Index, error) {
	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", dm)
	mapping.DefaultField = "_all"
	return bleve.NewMemOnly(mapping)
}
