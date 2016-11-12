package index

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzers/keyword_analyzer"
	"github.com/blevesearch/bleve/analysis/analyzers/simple_analyzer"
	"github.com/blevesearch/bleve/analysis/analyzers/standard_analyzer"
	"github.com/blevesearch/bleve/analysis/datetime_parsers/datetime_optional"
	"github.com/nhurel/dim/lib/registry"
	"github.com/nhurel/dim/lib/utils"
)

// Image modeling for indexation
type Image struct {
	ID           string
	Name         string
	FullName     string
	Tag          string
	Comment      string
	Created      time.Time
	Author       string
	Label        map[string]string
	Labels       []string
	Volumes      []string
	ExposedPorts []int
	Env          map[string]string
	Envs         []string
	Size         int64
	//Config *container.Config

}

// Type implementation of bleve.Classifier interface
func (im Image) Type() string {
	return "image"
}

// Parse converts a docker image into an indexable image
func Parse(name string, img *registry.Image) *Image {
	parsed := &Image{
		ID:      img.Digest,
		Name:    name,
		Tag:     img.Tag,
		Comment: img.Comment,
		Created: img.Created,
		Author:  img.Author,
	}
	parsed.FullName = fmt.Sprintf("%s:%s", name, img.Tag)

	parsed.Label = utils.FilterValues(img.Config.Labels, "")
	parsed.Labels = utils.Keys(parsed.Label)

	volumes := make([]string, 0, len(img.Config.Volumes))
	for v := range img.Config.Volumes {
		volumes = append(volumes, v)
	}
	parsed.Volumes = volumes

	envs := make(map[string]string, len(img.Config.Env))
	for _, iLabel := range img.Config.Env {
		split := strings.Split(iLabel, "=") // TODO Use regexp for better label handling
		if len(split) > 1 {
			envs[split[0]] = split[1]
		}
	}
	parsed.Env = envs
	parsed.Envs = utils.Keys(envs)

	ports := make([]int, 0, len(img.Config.ExposedPorts))
	for p := range img.Config.ExposedPorts {
		ports = append(ports, p.Int())
	}
	parsed.ExposedPorts = ports

	parsed.Size = img.Size

	logrus.WithField("image", parsed).Debugln("Docker image parsed")
	return parsed
}

// ImageMapping is a document mapping that specifies how to index an image
var ImageMapping *bleve.DocumentMapping

func init() {

	ImageMapping = bleve.NewDocumentMapping()

	tagMapping := bleve.NewTextFieldMapping()
	tagMapping.Analyzer = keyword_analyzer.Name
	tagMapping.IncludeInAll = true
	tagMapping.Store = true
	ImageMapping.AddFieldMappingsAt("Tag", tagMapping)

	nameMapping := bleve.NewTextFieldMapping()
	nameMapping.Analyzer = simple_analyzer.Name
	nameMapping.IncludeInAll = true
	nameMapping.Store = true
	ImageMapping.AddFieldMappingsAt("Name", nameMapping)
	ImageMapping.AddFieldMappingsAt("FullName", nameMapping)

	idMapping := bleve.NewTextFieldMapping()
	idMapping.Analyzer = keyword_analyzer.Name
	idMapping.Store = true
	idMapping.IncludeInAll = false
	idMapping.Index = true
	ImageMapping.AddFieldMappingsAt("ID", idMapping)

	authorMapping := bleve.NewTextFieldMapping()
	authorMapping.Analyzer = simple_analyzer.Name
	authorMapping.IncludeInAll = false
	authorMapping.Store = true
	ImageMapping.AddFieldMappingsAt("Author", authorMapping)
	ImageMapping.AddFieldMappingsAt("Volumes", authorMapping)
	ImageMapping.AddFieldMappingsAt("Labels", authorMapping)
	ImageMapping.AddFieldMappingsAt("Labels", idMapping)
	ImageMapping.AddFieldMappingsAt("Label", authorMapping)
	ImageMapping.AddFieldMappingsAt("Envs", authorMapping)
	ImageMapping.AddFieldMappingsAt("Envs", idMapping)
	ImageMapping.AddFieldMappingsAt("Env", authorMapping)

	commentMapping := bleve.NewTextFieldMapping()
	commentMapping.Analyzer = standard_analyzer.Name
	commentMapping.IncludeInAll = true
	commentMapping.Store = true
	ImageMapping.AddFieldMappingsAt("Comment", commentMapping)

	dateMapping := bleve.NewDateTimeFieldMapping()
	dateMapping.DateFormat = datetime_optional.Name
	dateMapping.Store = true
	dateMapping.IncludeInAll = false
	ImageMapping.AddFieldMappingsAt("Created", dateMapping)

	portsMapping := bleve.NewNumericFieldMapping()
	portsMapping.Store = true
	portsMapping.IncludeInAll = false
	ImageMapping.AddFieldMappingsAt("ExposedPorts", portsMapping)
	ImageMapping.AddFieldMappingsAt("Size", portsMapping)

	ImageMapping.DefaultAnalyzer = simple_analyzer.Name

}
