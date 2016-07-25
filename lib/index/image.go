package index

import (
	"github.com/blevesearch/bleve/analysis/analyzers/simple_analyzer"
	"github.com/blevesearch/bleve/analysis/datetime_parsers/datetime_optional"
	"github.com/blevesearch/blevex/detect_lang"
	"time"
	//"github.com/docker/engine-api/types/container"
	"github.com/Sirupsen/logrus"
	"github.com/blevesearch/bleve"
	"github.com/docker/docker/image"
	"strings"
)

type Image struct {
	ID           string
	Name         string
	Tag          string
	Comment      string
	Created      time.Time
	Author       string
	Labels       map[string]interface{}
	Volumes      []string
	ExposedPorts []int
	Env          map[string]string
	//Config *container.Config

}

// Implement bleve.Classifier interface
func (im Image) Type() string {
	return "image"
}

// Parse converts a docker image into an indexable image
func Parse(id, name, tag string, img *image.Image) *Image {
	parsed := &Image{
		ID:      id,
		Name:    name,
		Tag:     tag,
		Comment: img.Comment,
		Created: img.Created,
		Author:  img.Author,
	}
	labels := make(map[string]interface{}, len(img.Config.Labels))
	for _, iLabel := range img.Config.Labels {
		split := strings.Split(iLabel, "=") // TODO Use regexp for better label handling
		if len(split) > 1 {
			labels[split[0]] = split[1]
		} else {
			labels[split[0]] = true
		}
	}
	parsed.Labels = labels

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

	logrus.WithField("image", parsed).Debugln("Docker image parsed")
	return parsed
}

var imageMapping *bleve.DocumentMapping

func init() {

	imageMapping = bleve.NewDocumentMapping()

	tagMapping := bleve.NewTextFieldMapping()
	tagMapping.Analyzer = simple_analyzer.Name
	tagMapping.IncludeInAll = true
	tagMapping.Store = true

	imageMapping.AddFieldMappingsAt("Tag", tagMapping)
	imageMapping.AddFieldMappingsAt("Name", tagMapping)

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
	commentMapping.Analyzer = detect_lang.AnalyzerName
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



	imageMapping.DefaultAnalyzer = simple_analyzer.Name

	// FIXME: how should be indexed collections ?

}
