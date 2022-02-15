package syaml

import (
	"github.com/tidwall/sjson"
	"sigs.k8s.io/yaml"
)

// SetBytes accepts a YAML body, a path and a new value, and updates the
// specific key in the YAML body using the path.
// See https://github.com/tidwall/sjson#examples
func SetBytes(y []byte, path string, value interface{}) ([]byte, error) {
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return nil, err
	}
	updated, err := sjson.SetBytes(j, path, value)
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(updated)
}

// DeleteBytes accepts a YAML body and a path, and deletes the
// specific key in the YAML body matching the path.
// See https://github.com/tidwall/sjson#examples
func DeleteBytes(y []byte, path string) ([]byte, error) {
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return nil, err
	}
	updated, err := sjson.DeleteBytes(j, path)
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(updated)
}
