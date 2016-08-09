package index

import (
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzers/keyword_analyzer"
	"github.com/blevesearch/bleve/analysis/analyzers/simple_analyzer"
	"github.com/blevesearch/bleve/analysis/analyzers/standard_analyzer"
	"github.com/blevesearch/bleve/analysis/datetime_parsers/datetime_optional"
	"github.com/nhurel/dim/lib/registry"
	"strings"
	"time"
)

type Image struct {
	ID           string
	Name         string
	Tag          string
	Comment      string
	Created      time.Time
	Author       string
	Labels       map[string]string
	Volumes      []string
	ExposedPorts []int
	Env          map[string]string
	Size         int64
	//Config *container.Config

}

// Implement bleve.Classifier interface
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

	parsed.Labels = img.Config.Labels

	volumes := make([]string, 0, len(img.Config.Volumes))
	for v, _ := range img.Config.Volumes {
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

	ports := make([]int, 0, len(img.Config.ExposedPorts))
	for p, _ := range img.Config.ExposedPorts {
		ports = append(ports, p.Int())
	}
	parsed.ExposedPorts = ports

	parsed.Size = img.Size

	logrus.WithField("image", parsed).Debugln("Docker image parsed")
	return parsed
}

var imageMapping *bleve.DocumentMapping

func init() {

	imageMapping = bleve.NewDocumentMapping()

	tagMapping := bleve.NewTextFieldMapping()
	tagMapping.Analyzer = keyword_analyzer.Name
	tagMapping.IncludeInAll = true
	tagMapping.Store = true
	imageMapping.AddFieldMappingsAt("Tag", tagMapping)

	nameMapping := bleve.NewTextFieldMapping()
	nameMapping.Analyzer = simple_analyzer.Name
	nameMapping.IncludeInAll = true
	nameMapping.Store = true
	imageMapping.AddFieldMappingsAt("Name", nameMapping)

	disabledFieldMapping := bleve.NewTextFieldMapping()
	disabledFieldMapping.Store = false
	disabledFieldMapping.IncludeInAll = false
	disabledFieldMapping.Index = false
	imageMapping.AddFieldMappingsAt("ID", disabledFieldMapping)

	authorMapping := bleve.NewTextFieldMapping()
	authorMapping.Analyzer = simple_analyzer.Name
	authorMapping.IncludeInAll = false
	authorMapping.Store = true
	imageMapping.AddFieldMappingsAt("Author", authorMapping)
	imageMapping.AddFieldMappingsAt("Volumes", authorMapping)
	imageMapping.AddFieldMappingsAt("Labels", authorMapping)

	commentMapping := bleve.NewTextFieldMapping()
	commentMapping.Analyzer = standard_analyzer.Name
	commentMapping.IncludeInAll = true
	commentMapping.Store = true
	imageMapping.AddFieldMappingsAt("Comment", commentMapping)

	dateMapping := bleve.NewDateTimeFieldMapping()
	dateMapping.DateFormat = datetime_optional.Name
	dateMapping.Store = true
	dateMapping.IncludeInAll = false
	imageMapping.AddFieldMappingsAt("Created", dateMapping)

	portsMapping := bleve.NewNumericFieldMapping()
	portsMapping.Store = false
	portsMapping.IncludeInAll = false
	imageMapping.AddFieldMappingsAt("ExposedPorts", portsMapping)
	imageMapping.AddFieldMappingsAt("Size", portsMapping)

	imageMapping.DefaultAnalyzer = simple_analyzer.Name

	// FIXME: how should be indexed collections ?

}
