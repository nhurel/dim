package dim

import (
	"fmt"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"testing"
)

func TestAddLabel(t *testing.T) {
	// GIVEN
	d := &Dim{Docker: &NoOpDockerClient{}}
	parent := "parentImage:test"
	tests := []struct {
		labels   []string
		expected error
	}{
		{labels: []string{}, expected: fmt.Errorf("No label provided")},
		{labels: []string{""}, expected: fmt.Errorf("Failed to parse given label ")},
		{labels: []string{"label1=value1"}},
		{labels: []string{"label1NoValue"}, expected: fmt.Errorf("Failed to parse given label label1NoValue")},
		{labels: []string{"label1=value1", "label2=value2"}},
		{labels: []string{"label1=value1", "label2NoValue"}, expected: fmt.Errorf("Failed to parse given label label2NoValue")},
	}

	for _, test := range tests {
		//WHEN
		err := d.AddLabel(parent, test.labels, "")

		//THEN
		if (err == nil && test.expected != nil) || (err != nil && test.expected == nil) || (err != nil && err.Error() != test.expected.Error()) {
			t.Errorf("Wrong error returned when label is %v. Got %v - Expected %v", test.labels, err, test.expected)
		}
	}
}

type NoOpDockerClient struct {
}

func (*NoOpDockerClient) ImageBuild(parent string, buildLabels map[string]string, tag string) error {
	return nil
}
func (*NoOpDockerClient) Pull(image string) error {
	return nil
}
func (*NoOpDockerClient) Inspect(image string) (types.ImageInspect, error) {
	return types.ImageInspect{ContainerConfig: &container.Config{}}, nil
}

func (*NoOpDockerClient) Remove(image string) error {
	return nil
}
