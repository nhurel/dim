package indextest

import (
	"path"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/nhurel/dim/lib/index"
)

// ParseTime converts a given string into time
func ParseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

// MockIndex creates a bleve Index for tests
func MockIndex() (bleve.Index, error) {
	logrus.SetLevel(logrus.InfoLevel)
	dir := path.Join("test.index", time.Now().Format("20060102150405.000"))

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("image", index.ImageMapping)
	mapping.DefaultField = "_all"
	return bleve.New(dir, mapping)
}
